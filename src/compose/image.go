package compose

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

func NewImageNameFromString(image string) *ImageName {
	dockerImage := &ImageName{}
	split := strings.SplitN(image, ":", 2)
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

// Type structures used for cleaning images
// Able to sort out old tags by creation date

type imageTags struct {
	images []*imageTag
}

type imageTag struct {
	id      string
	name    ImageName
	created int64
}

func (tags *imageTags) Len() int {
	return len(tags.images)
}

func (tags *imageTags) Less(i, j int) bool {
	return tags.images[i].created > tags.images[j].created
}

func (tags *imageTags) Swap(i, j int) {
	tags.images[i], tags.images[j] = tags.images[j], tags.images[i]
}

func (tags *imageTags) getOld(keep int) []ImageName {
	if len(tags.images) < keep {
		return nil
	}

	sort.Sort(tags)

	result := []ImageName{}
	for i := keep; i < len(tags.images); i++ {
		result = append(result, tags.images[i].name)
	}

	return result
}
