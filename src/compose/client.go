package compose

import (
	"fmt"
	"io"
	"time"
	"util"

	"github.com/kr/pretty"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/fsouza/go-dockerclient"
)

type Client interface {
	GetContainers() ([]*Container, error)
	RemoveContainer(container *Container) error
	RunContainer(container *Container) error
	EnsureContainerExist(name *Container) error
	EnsureContainerState(name *Container) error
	PullAll(config *Config) error
	AttachToContainers(container []*Container) error
	AttachToContainer(container *Container) error
	FetchImages(containers []*Container) error
	WaitForContainer(container *Container) error
}

type ClientCfg struct {
	Docker *docker.Client
	Global bool // Search for existing containers globally, not only ones started with compose
	Attach bool
	Wait   time.Duration
	Auth   *AuthConfig
}

type ErrContainerBadState struct {
	Container  *Container
	Running    bool
	OOMKilled  bool
	ExitCode   int
	ErrorStr   string
	StartedAt  time.Time
	FinishedAt time.Time
}

type AuthConfig struct {
	Username      string
	Password      string
	Email         string
	ServerAddress string
}

func (a *AuthConfig) ToDockerApi() *docker.AuthConfiguration {
	if a == nil {
		return &docker.AuthConfiguration{}
	}
	return &docker.AuthConfiguration{
		Username:      a.Username,
		Password:      a.Password,
		Email:         a.Email,
		ServerAddress: a.ServerAddress,
	}
}

func (e ErrContainerBadState) Error() string {
	str := fmt.Sprintf("Container %s exited with code %d", e.Container.Name, e.ExitCode)
	if e.ErrorStr != "" {
		str = fmt.Sprintf("%s, error: %s", str, e.ErrorStr)
	}
	return str
}

func NewClient(initialClient *ClientCfg) (Client, error) {
	client := &ClientCfg{
		Docker: initialClient.Docker,
		Global: initialClient.Global,
		Attach: initialClient.Attach,
		Wait:   initialClient.Wait,
		Auth:   initialClient.Auth,
	}
	return client, nil
}

func (client *ClientCfg) GetContainers() ([]*Container, error) {
	filters := map[string][]string{}
	if !client.Global {
		filters["label"] = []string{"rocker-compose-id"}
	}

	apiContainers, err := client.Docker.ListContainers(docker.ListContainersOptions{
		All:     true,
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	containers := []*Container{}

	if len(apiContainers) == 0 {
		return containers, nil
	}

	// Fetch detailed information about all containers in parallel
	type chResponse struct {
		container *docker.Container
		err       error
	}

	ch := make(chan *chResponse, len(apiContainers))
	numResponses := 0

	for _, apiContainer := range apiContainers {
		go func(apiContainer docker.APIContainers) {
			chResponse := new(chResponse)
			chResponse.container, chResponse.err = client.Docker.InspectContainer(apiContainer.ID)
			ch <- chResponse
		}(apiContainer)
	}

	log.Infof("Gathering info about %d containers", len(apiContainers))

	for {
		select {
		case resp := <-ch:
			numResponses++
			if resp.err != nil {
				return nil, fmt.Errorf("Failed to fetch container, error: %s", resp.err)
			}
			container, err := NewContainerFromDocker(resp.container)
			if err != nil {
				return nil, fmt.Errorf("Failed to initialize config container instance from docker api, error: %s", err)
			}
			containers = append(containers, container)

		case <-time.After(30 * time.Second):
			// todo: you may have to use client.Timeout
			return nil, fmt.Errorf("Timeout while fetching containers")
		}

		if len(apiContainers) == numResponses {
			break
		}
	}

	return containers, nil
}

func (client *ClientCfg) RemoveContainer(container *Container) error {
	log.Infof("Removing container %s id:%s", container.Name, util.TruncateID(container.Id))

	if container.Config.KillTimeout != nil && *container.Config.KillTimeout > 0 {
		if err := client.Docker.StopContainer(container.Id, *container.Config.KillTimeout); err != nil {
			return fmt.Errorf("Failed to stop container, error: %s", err)
		}
	}
	keepVolumes := container.Config.KeepVolumes != nil && *container.Config.KeepVolumes
	removeOptions := docker.RemoveContainerOptions{
		ID:            container.Id,
		RemoveVolumes: !keepVolumes,
		Force:         true,
	}
	if err := client.Docker.RemoveContainer(removeOptions); err != nil {
		return fmt.Errorf("Failed to remove container, error: %s", err)
	}

	return nil
}

func (client *ClientCfg) RunContainer(container *Container) error {
	log.Infof("Create container %s", container.Name)

	opts, err := container.CreateContainerOptions()
	if err != nil {
		return fmt.Errorf("Failed to initialize container options, error: %s", err)
	}
	log.Debugf("Creating container with opts: %# v", pretty.Formatter(opts))

	apiContainer, err := client.Docker.CreateContainer(*opts)
	if err != nil {
		return fmt.Errorf("Failed to create container, error: %s", err)
	}
	container.Id = apiContainer.ID

	if container.State.Running {
		if client.Attach {
			if err := client.AttachToContainer(container); err != nil {
				return err
			}
		}

		log.Infof("Starting container %s id:%s", container.Name, util.TruncateID(container.Id))

		// TODO: HostConfig may be changed without re-creation of containers
		// so of Volumes or Links are changed, we just need to restart container
		if err := client.Docker.StartContainer(container.Id, container.Config.GetApiHostConfig()); err != nil {
			return fmt.Errorf("Failed to start container, error: %s", err)
		}

		if client.Wait > 0 {
			log.Infof("Waiting for %s to ensure %s not exited abnormally...", client.Wait, container.Name)
			time.Sleep(client.Wait)

			if err := client.EnsureContainerState(container); err != nil {
				// TODO: create container io once in some place?
				if !client.Attach {
					container.Io = NewContainerIo(container)

					err2 := client.Docker.Logs(docker.LogsOptions{
						Container:    container.Name.String(),
						OutputStream: container.Io.Stdout,
						ErrorStream:  container.Io.Stderr,
						Stdout:       true,
						Stderr:       true,
					})
					if err2 != nil {
						log.Errorf("Failed to read logs of container %s, error: %s", container.Name, err2)
					}
				}

				return err
			}
		}
	}

	return nil
}

func (client *ClientCfg) EnsureContainerExist(container *Container) error {
	log.Infof("Checking container exist %s", container.Name)
	if _, err := client.Docker.InspectContainer(container.Name.String()); err != nil {
		return err
	}
	return nil
}

func (client *ClientCfg) EnsureContainerState(container *Container) error {
	log.Debugf("Checking container state %s", container.Name)
	inspect, err := client.Docker.InspectContainer(container.Name.String())
	if err != nil {
		return err
	}
	err = ErrContainerBadState{
		Container:  container,
		Running:    inspect.State.Running,
		OOMKilled:  inspect.State.OOMKilled,
		ExitCode:   inspect.State.ExitCode,
		ErrorStr:   inspect.State.Error,
		StartedAt:  inspect.State.StartedAt,
		FinishedAt: inspect.State.FinishedAt,
	}
	log.Debugf("Container state for %s: %# v", container.Name, inspect.State)
	// container.State.Running = inspect.State.Running
	// if inspect.State.Running != container.State.Running {
	// 	return err
	// }
	if inspect.State.ExitCode != 0 {
		return err
	}
	return nil
}

func (client *ClientCfg) PullAll(config *Config) error {
	// TODO: do not pull same image twice
	for _, container := range config.GetContainers() {
		if err := client.pullImageForContainer(container); err != nil {
			return err
		}
	}
	return nil
}

func (client *ClientCfg) AttachToContainer(container *Container) error {
	success := make(chan struct{})
	errors := make(chan error, 1)

	container.Io = NewContainerIo(container)

	go func() {
		var err error
		defer container.Io.Done(err)

		err = client.Docker.AttachToContainer(docker.AttachToContainerOptions{
			Container:    container.Name.String(),
			OutputStream: container.Io.Stdout,
			ErrorStream:  container.Io.Stderr,
			Stdout:       true,
			Stderr:       true,
			Stream:       true,
			Success:      success,
		})
	}()

	select {
	case err := <-errors:
		return err
	case ack := <-success:
		success <- ack
	}

	return nil
}

func (client *ClientCfg) AttachToContainers(containers []*Container) error {
	running := []*Container{}
	for _, c := range containers {
		if c.State.Running {
			running = append(running, c)
		}
	}

	wg := util.NewErrorWaitGroup(len(running))

	for _, container := range running {
		if container.Io == nil {
			if err := client.AttachToContainer(container); err != nil {
				return err
			}
		}

		go func(container *Container) {
			wg.Done(container.Io.Wait())
		}(container)
	}

	return wg.Wait()
}

func (client *ClientCfg) WaitForContainer(container *Container) error {
	log.Infof("Waiting container to finish %s", container.Name)
	status, err := client.Docker.WaitContainer(container.Name.String())

	if err != nil {
		return err
	}

	if status != 0 {
		return fmt.Errorf("Non-zero exit code %d received from container %s", status, container.Name)
	}

	return nil
}


func (client *ClientCfg) FetchImages(containers []*Container) error {
	type message struct {
		container *Container
		result    chan error
	}

	wg := util.NewErrorWaitGroup(len(containers))
	chPullImages := make(chan message)
	done := make(chan struct{}, 1)

	// Pull worker
	// We do not want to pull images in parallel
	// instead, we use an actor to pull images sequentially
	go func() {
		for {
			select {
			case msg := <-chPullImages:
				msg.result <- client.pullImageForContainer(msg.container)
			case <-done:
				return
			}
		}
	}()

	defer func() {
		done <- struct{}{}
	}()

	for _, container := range containers {
		if container.Image == nil {
			return fmt.Errorf("Image is not specified for container %s", container.Name)
		}
		go func(container *Container) {
			image, err := client.Docker.InspectImage(container.Image.String())
			if err == docker.ErrNoSuchImage {
				msg := message{container, make(chan error)}
				chPullImages <- msg
				wg.Done(<-msg.result)
				return
			} else if err != nil {
				wg.Done(err)
				return
			}
			container.ImageId = image.ID
			wg.Done(nil)
		}(container)
	}

	return wg.Wait()
}

// Internal

func (client *ClientCfg) pullImageForContainer(container *Container) error {
	if container.Image == nil {
		return fmt.Errorf("Image is not specified for container: %s", container.Name)
	}

	log.Infof("Pulling image: %s for %s", container.Image, container.Name)

	pipeReader, pipeWriter := io.Pipe()

	pullOpts := docker.PullImageOptions{
		Repository:    container.Image.NameWithRegistry(),
		Registry:      container.Image.Registry,
		Tag:           container.Image.Tag,
		OutputStream:  pipeWriter,
		RawJSONStream: true,
	}

	//fmt.Fprintf(builder.OutStream, " ===> docker pull %s\n", container.Image)

	errch := make(chan error, 1)

	go func() {
		err := client.Docker.PullImage(pullOpts, *client.Auth.ToDockerApi())

		if err := pipeWriter.Close(); err != nil {
			log.Errorf("Failed to close pull image stream for %s, error: %s", container.Name, err)
		}

		errch <- err
	}()

	def := log.StandardLogger()
	fd, isTerminal := term.GetFdInfo(def.Out)
	out := def.Out

	if !isTerminal {
		out = def.Writer()
	}

	if err := jsonmessage.DisplayJSONMessagesStream(pipeReader, out, fd, isTerminal); err != nil {
		return fmt.Errorf("Failed to process json stream for image: %s, error: %s", container.Image, err)
	}

	if err := <-errch; err != nil {
		return fmt.Errorf("Failed to pull image %s for container %s, error: %s", container.Image, container.Name, err)
	}

	return nil
}
