package imagename

import (
	"github.com/wmark/semver"
	"sort"
	"strings"
)

const (
	Latest    = "latest"
	Wildcards = "x*"
)

type ImageName struct {
	Registry string
	Name     string
	Tag      string
	Version  *semver.Range
}

func (dockerImage ImageName) GetTag() string {
	if dockerImage.HasTag() {
		return dockerImage.Tag
	}
	return Latest
}

func (dockerImage ImageName) HasTag() bool {
	return dockerImage.Tag != "" && !strings.ContainsAny(dockerImage.Tag, Wildcards)
}

func (dockerImage ImageName) HasVersion() bool {
	return dockerImage.Version != nil
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

func NewFromString(image string) *ImageName {
	split := strings.SplitN(image, ":", 2)
	if len(split) > 1 {
		return New(split[0], split[1])
	}
	return New(split[0], "")
}

func New(image string, tag string) *ImageName {
	dockerImage := &ImageName{}
	if strings.Contains(image, ".") || len(strings.SplitN(image, "/", 3)) > 2 {
		registryAndName := strings.SplitN(image, "/", 2)
		dockerImage.Registry = registryAndName[0]
		dockerImage.Name = registryAndName[1]
	} else {
		dockerImage.Name = image
	}
	if tag != "" {
		if rng, err := semver.NewRange(tag); err == nil && rng != nil {
			dockerImage.Version = rng
		}
		if ver, err := semver.NewVersion(strings.TrimLeft(tag, "v")); (err == nil && ver != nil) || dockerImage.Version == nil || strings.ContainsAny(tag, Wildcards) {
			dockerImage.Tag = tag
		}
	}
	return dockerImage
}

func (dockerImage ImageName) Contains(b *ImageName) bool {
	if b == nil {
		return false
	}

	if dockerImage.Name != b.Name || dockerImage.Registry != b.Registry {
		return false
	}

	if strings.ContainsAny(dockerImage.Tag, Wildcards) {
		return true
	}

	if dockerImage.HasTag() && dockerImage.Tag == b.Tag {
		return true
	}

	if dockerImage.HasVersion() && dockerImage.Version.IsSatisfiedBy(b.TagAsVersion()) {
		return true
	}

	return !dockerImage.HasTag() && !dockerImage.HasVersion()
}

func (dockerImage ImageName) TagAsVersion() (ver *semver.Version) {
	if !dockerImage.HasTag() {
		return nil
	}
	ver, _ = semver.NewVersion(dockerImage.Tag)
	return
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
