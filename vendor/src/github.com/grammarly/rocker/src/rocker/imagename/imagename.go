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
	if dockerImage.IsStrict() {
		return dockerImage.Tag
	}
	return Latest
}

func (dockerImage ImageName) IsStrict() bool {
	if dockerImage.HasVersionRange() {
		return dockerImage.TagAsVersion() != nil
	}
	return dockerImage.Tag != ""
}

func (dockerImage ImageName) All() bool {
	return strings.Contains(Wildcards, dockerImage.Tag)
}

func (dockerImage ImageName) HasVersion() bool {
	return dockerImage.TagAsVersion() != nil
}

func (dockerImage ImageName) HasVersionRange() bool {
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
		dockerImage.Tag = tag
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

	// semver library has a bug with wildcards, so this checks are
	// necessary: empty range (or wildcard range) cannot contains any version, it just fails
	if dockerImage.All() {
		return true
	}

	if dockerImage.IsStrict() && dockerImage.Tag == b.Tag {
		return true
	}

	if dockerImage.HasVersionRange() && b.HasVersion() && dockerImage.Version.IsSatisfiedBy(b.TagAsVersion()) {
		return true
	}

	return dockerImage.Tag == "" && !dockerImage.HasVersionRange()
}

func (dockerImage ImageName) TagAsVersion() (ver *semver.Version) {
	ver, _ = semver.NewVersion(strings.TrimPrefix(dockerImage.Tag, "v"))
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
