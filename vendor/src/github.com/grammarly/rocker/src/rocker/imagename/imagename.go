/*-
 * Copyright 2015 Grammarly, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package imagename provides docker data structure for docker image names
// It also provides a number of utility functions, related to image name resolving,
// comparison, and semver functionality.
package imagename

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/wmark/semver"
)

const (
	// Latest is :latest tag value
	Latest = "latest"

	// Wildcards is wildcard value variants
	Wildcards = "x*"
)

// ImageName is the data structure with describes docker image name
type ImageName struct {
	Registry string
	Name     string
	Tag      string
	Version  *semver.Range
}

// NewFromString parses a given string and returns ImageName
func NewFromString(image string) *ImageName {
	split := strings.SplitN(image, ":", 2)
	if len(split) > 1 {
		return New(split[0], split[1])
	}
	return New(split[0], "")
}

// New parses a given 'image' and 'tag' strings and returns ImageName
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

// String returns the string representation of the current image name
func (img ImageName) String() string {
	return img.NameWithRegistry() + ":" + img.GetTag()
}

// GetTag returns the tag of the current image name
func (img ImageName) GetTag() string {
	return img.Tag
}

// IsStrict returns true if tag of the current image is specified and contains no fuzzy rules
func (img ImageName) IsStrict() bool {
	if img.HasVersionRange() {
		return img.TagAsVersion() != nil
	}
	return img.Tag != ""
}

// All returns true if tag of the current image refers to any version
func (img ImageName) All() bool {
	return strings.Contains(Wildcards, img.Tag)
}

// HasVersion returns true if tag of the current image refers to a semver format
func (img ImageName) HasVersion() bool {
	return img.TagAsVersion() != nil
}

// HasVersionRange returns true if tag of the current image refers to a semver format and is fuzzy
func (img ImageName) HasVersionRange() bool {
	return img.Version != nil
}

// TagAsVersion return semver.Version of the current image tag in case it's in semver format
func (img ImageName) TagAsVersion() (ver *semver.Version) {
	ver, _ = semver.NewVersion(strings.TrimPrefix(img.Tag, "v"))
	return
}

// IsSameKind returns true if current image and the given one are same but may have different versions (tags)
func (img ImageName) IsSameKind(b ImageName) bool {
	return img.Registry == b.Registry && img.Name == b.Name
}

// NameWithRegistry returns the [registry/]name of the current image name
func (img ImageName) NameWithRegistry() string {
	registryPrefix := ""
	if img.Registry != "" {
		registryPrefix = img.Registry + "/"
	}
	return registryPrefix + img.Name
}

// Contains returns true if the current image tag wildcard satisfies a given image version
func (img ImageName) Contains(b *ImageName) bool {
	if b == nil {
		return false
	}

	if !img.IsSameKind(*b) {
		return false
	}

	// semver library has a bug with wildcards, so this checks are
	// necessary: empty range (or wildcard range) cannot contain any version, it just fails
	if img.All() {
		return true
	}

	if img.IsStrict() && img.Tag == b.Tag {
		return true
	}

	if img.HasVersionRange() && b.HasVersion() && img.Version.IsSatisfiedBy(b.TagAsVersion()) {
		return true
	}

	return img.Tag == "" && !img.HasVersionRange()
}

// UnmarshalJSON parses JSON string and returns ImageName
func (img *ImageName) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*img = *NewFromString(s)
	return nil
}

// MarshalJSON serializes ImageName to JSON string
func (img ImageName) MarshalJSON() ([]byte, error) {
	return json.Marshal(img.String())
}

// Tags is a structure used for cleaning images
// Sorts out old tags by creation date
type Tags struct {
	Items []*Tag
}

// Tag is a structure used for cleaning images
type Tag struct {
	ID      string
	Name    ImageName
	Created int64
}

// Len returns the length of image tags
func (tags *Tags) Len() int {
	return len(tags.Items)
}

// Less returns true if item by index[i] is created after of item[j]
func (tags *Tags) Less(i, j int) bool {
	return tags.Items[i].Created > tags.Items[j].Created
}

// Swap swaps items by indices [i] and [j]
func (tags *Tags) Swap(i, j int) {
	tags.Items[i], tags.Items[j] = tags.Items[j], tags.Items[i]
}

// GetOld returns the list of items older then [keep] newest items sorted by Created date
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
