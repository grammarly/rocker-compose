package imagename

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/fsouza/go-dockerclient"
)

type tags struct {
	Name string   `json:"name,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

type history struct {
	Compatibility string `json:"v1Compatibility,omitempty"`
}

type manifests struct {
	Name          string     `json:"name,omitempty"`
	Tag           string     `json:"tag,omitempty"`
	Architecture  string     `json:"architecture,omitempty"`
	History       []*history `json:"history,omitempty"`
	SchemaVersion int        `json:"schemaVersion,omitempty"`
}

type DockerHub struct{}

func NewDockerHub() *DockerHub {
	return &DockerHub{}
}

func (h *DockerHub) Get(image *ImageName) (img *docker.Image, err error) {
	manifest := manifests{}
	img = &docker.Image{}
	if err = h.doGet(fmt.Sprintf("https://%s/v2/%s/manifests/%s", image.Registry, image.Name, image.Tag), &manifest); err != nil {
		return
	}

	if len(manifest.History) == 0 {
		err = fmt.Errorf("Image doesn't have expected format, history record is empty")
		return
	}

	last := manifest.History[0]
	err = json.Unmarshal([]byte(last.Compatibility), img)
	return
}

func (h *DockerHub) List(image *ImageName) (images []*ImageName, err error) {
	tg := tags{}
	if err = h.doGet(fmt.Sprintf("https://%s/v2/%s/tags/list", image.Registry, image.Name), &tg); err != nil {
		return
	}
	for _, t := range tg.Tags {
		candidate := New(image.NameWithRegistry(), t)
		if image.Contains(candidate) {
			images = append(images, candidate)
		}
	}
	return
}

func (h *DockerHub) doGet(url string, obj interface{}) (err error) {
	var res *http.Response
	var body []byte

	res, err = http.Get(url)
	if err != nil {
		err = fmt.Errorf("Request to %s failed with %s\n", url, err)
		return
	}

	if res.StatusCode == 404 {
		err = fmt.Errorf("Not found")
		return
	}

	if body, err = ioutil.ReadAll(res.Body); err != nil {
		err = fmt.Errorf("Response from %s cannot be read due to error %s\n", url, err)
		return
	}

	if err = json.Unmarshal(body, obj); err != nil {
		err = fmt.Errorf("Response from %s cannot be unmarshalled due to error %s, response: %s\n",
			url, err, string(body))
		return
	}

	return
}
