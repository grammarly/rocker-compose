package compose

import (
	"strings"
)

type ImageName struct {
	Registry string
	Name     string
	Tag      string
}

func (dockerImage *ImageName) NameWithRegistry() string {
	registryPrefix := ""
	if dockerImage.Registry != "" {
		registryPrefix = dockerImage.Registry + "/"
	}
	return registryPrefix + dockerImage.Name
}

func (dockerImage ImageName) String() string {
	return dockerImage.NameWithRegistry() + ":" + dockerImage.Tag
}

func NewImageNameFromString(image string) *ImageName {
	dockerImage := &ImageName{}
	split := strings.SplitN(image, ":", 2)
	// TODO: do we allow dots "." in image name?
	// if yes, find another way to distinguish registry from account name part
	// Maybe, borrow this: https://github.com/jwilder/docker-gen/blob/82da74aaa33dbc9bf96ebf53a11ce289e672dbf2/docker_client.go#L77
	if strings.Contains(split[0], ".") {
		registryAndName := strings.SplitN(split[0], "/", 2)
		dockerImage.Registry = registryAndName[0]
		dockerImage.Name = registryAndName[1]
	} else {
		dockerImage.Name = split[0]
	}
	if len(split) == 1 {
		dockerImage.Tag = "latest"
	} else {
		dockerImage.Tag = split[1]
	}
	return dockerImage
}
