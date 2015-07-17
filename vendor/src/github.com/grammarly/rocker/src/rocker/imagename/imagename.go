package imagename

import (
	"sort"
	"strings"
)

const Latest = "latest"

type ImageName struct {
	Registry string
	Name     string
	Tag      string
}

func (dockerImage ImageName) GetTag() string {
	if dockerImage.HasTag() {
		return dockerImage.Tag
	}
	return Latest
}

func (dockerImage ImageName) HasTag() bool {
	return dockerImage.Tag != ""
}

func (a ImageName) IsSameKind(b ImageName) bool {
	return a.Registry == b.Registry && a.Name == b.Name
}

func (dockerImage ImageName) NameWithRegistry() string {
	registryPrefix := ""
	if dockerImage.Registry != "" {
		registryPrefix = dockerImage.Registry + "/"
	}
	return registryPrefix + dockerImage.Name
}

func (dockerImage ImageName) String() string {
	return dockerImage.NameWithRegistry() + ":" + dockerImage.GetTag()
}

func New(image string) *ImageName {
	dockerImage := &ImageName{}
	split := strings.SplitN(image, ":", 2)
	if strings.Contains(split[0], ".") || len(strings.SplitN(image, "/", 3)) > 2 {
		registryAndName := strings.SplitN(split[0], "/", 2)
		dockerImage.Registry = registryAndName[0]
		dockerImage.Name = registryAndName[1]
	} else {
		dockerImage.Name = split[0]
	}
	if len(split) > 1 {
		dockerImage.Tag = split[1]
	}
	return dockerImage
}

// Type structures used for cleaning images
// Able to sort out old tags by creation date
type Tags struct {
	Items []*Tag
}

type Tag struct {
	Id      string
	Name    ImageName
	Created int64
}

func (tags *Tags) Len() int {
	return len(tags.Items)
}

func (tags *Tags) Less(i, j int) bool {
	return tags.Items[i].Created > tags.Items[j].Created
}

func (tags *Tags) Swap(i, j int) {
	tags.Items[i], tags.Items[j] = tags.Items[j], tags.Items[i]
}

func (tags *Tags) GetOld(keep int) []ImageName {
	if len(tags.Items) < keep {
		return nil
	}

	sort.Sort(tags)

	result := []ImageName{}
	for i := keep; i < len(tags.Items); i++ {
		result = append(result, tags.Items[i].Name)
	}

	return result
}
