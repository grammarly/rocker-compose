package config

import (
	"fmt"
	"strings"
	"syscall"
)

var NotifySignals = map[string]syscall.Signal{
	"ABRT": syscall.SIGABRT,
	"ALRM": syscall.SIGALRM,
	"HUP":  syscall.SIGHUP,
	"INT":  syscall.SIGINT,
	"IO":   syscall.SIGIO,
	"KILL": syscall.SIGKILL,
	"QUIT": syscall.SIGQUIT,
	"STOP": syscall.SIGSTOP,
	"TERM": syscall.SIGTERM,
	"TRAP": syscall.SIGTRAP,
	"USR1": syscall.SIGUSR1,
	"USR2": syscall.SIGUSR2,
}

type Notify []NotifyAction

type NotifyAction interface {
	String() string
	// Run(client *docker.Client) error
	ContainerName() *ContainerName
}

type NotifyActionRestart struct {
	Container *ContainerName
}

type NotifyActionRecreate struct {
	Container *ContainerName
}

type NotifyActionKill struct {
	Container *ContainerName
	Signal    syscall.Signal
}

type NotifyActionExec struct {
	Container *ContainerName
	Cmd       []string
}

func NewNotifyAction(str string) (NotifyAction, error) {
	parts := strings.Split(str, " ")
	// if len(parts) < 2 {
	// 	return nil, fmt.Errorf("Bad notify spec, should be: `containerName [restart|reload|recreate|kill|exec] [args]`, got: `%s`", str)
	// }

	containerName := NewContainerNameFromString(parts[0])
	cmd := "restart"

	if len(parts) > 1 {
		cmd = parts[1]
	}

	switch cmd {
	case "restart":
		return &NotifyActionRestart{containerName}, nil

	case "reload":
		return &NotifyActionKill{containerName, syscall.SIGHUP}, nil

	case "recreate":
		return &NotifyActionRecreate{containerName}, nil

	case "kill":
		if len(parts) < 3 {
			return nil, fmt.Errorf("Bad notify kill spec, should be: `container_name kill -SIGNAL`, got: `%s`", str)
		}
		sigStr := strings.TrimPrefix(parts[2], "-")
		sig, ok := NotifySignals[sigStr]
		if !ok {
			signals := make([]string, 0, len(NotifySignals))
			for k := range NotifySignals {
				signals = append(signals, k)
			}
			return nil, fmt.Errorf("Bad notify kill signal `%s` in `%s`, available signals: %s", sigStr, str, strings.Join(signals, ", "))
		}
		return &NotifyActionKill{containerName, sig}, nil

	case "exec":
		if len(parts) < 3 {
			return nil, fmt.Errorf("Bad notify exec spec, should be: `container_name exec [cmd_args]`, got: `%s`", str)
		}
		return &NotifyActionExec{containerName, parts[2:]}, nil
	}

	return nil, fmt.Errorf("Bad notify command, available commands are: restart, kill, exec; got: `%s`", str)
}

func (n *NotifyActionRestart) String() string {
	return fmt.Sprintf("%s restart", n.Container)
}

func (n *NotifyActionRecreate) String() string {
	return fmt.Sprintf("%s recreate", n.Container)
}

func (n *NotifyActionKill) String() string {
	var sig string
	for k, v := range NotifySignals {
		if v == n.Signal {
			sig = k
			break
		}
	}
	return fmt.Sprintf("%s kill -%s", n.Container, sig)
}

func (n *NotifyActionExec) String() string {
	// TODO: support cmd with spaces
	return fmt.Sprintf("%s exec %s", n.Container, strings.Join(n.Cmd, " "))
}

func (n *NotifyActionRestart) ContainerName() *ContainerName {
	return n.Container
}

func (n *NotifyActionRecreate) ContainerName() *ContainerName {
	return n.Container
}

func (n *NotifyActionKill) ContainerName() *ContainerName {
	return n.Container
}

func (n *NotifyActionExec) ContainerName() *ContainerName {
	return n.Container
}
