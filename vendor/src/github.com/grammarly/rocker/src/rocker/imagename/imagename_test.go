/*-
 * Copyright 2014 Grammarly, Inc.
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

package imagename

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestImageParsingWithoutNamespace(t *testing.T) {
	img := New("repo/name:1")
	assert.Equal(t, "", img.Registry)
	assert.Equal(t, "1", img.Tag)
	assert.Equal(t, "repo/name", img.Name)
}

func TestImageRealLifeNamingExample(t *testing.T) {
	img := New("dockerhub.grammarly.io/platform/dockerize:v0.0.1")
	assert.Equal(t, "dockerhub.grammarly.io", img.Registry)
	assert.Equal(t, "platform/dockerize", img.Name)
	assert.Equal(t, "v0.0.1", img.Tag)
}

func TestImageRealLifeNamingExampleWithCapi(t *testing.T) {
	img := New("dockerhub.grammarly.io/common-api")
	assert.Equal(t, "dockerhub.grammarly.io", img.Registry)
	assert.Equal(t, "common-api", img.Name)
	assert.Equal(t, "latest", img.GetTag())
}

func TestImageParsingWithNamespace(t *testing.T) {
	img := New("hub/ns/name:1")
	assert.Equal(t, "hub", img.Registry)
	assert.Equal(t, "ns/name", img.Name)
	assert.Equal(t, "1", img.Tag)
}

func TestImageParsingWithoutTag(t *testing.T) {
	img := New("repo/name")
	assert.Equal(t, "", img.Registry)
	assert.Equal(t, "repo/name", img.Name)
	assert.Empty(t, img.Tag)
	assert.Equal(t, "latest", img.GetTag())
}

func TestImageLatest(t *testing.T) {
	img := New("rocker-build:latest")
	assert.Equal(t, "", img.Registry, "bag registry value")
	assert.Equal(t, "rocker-build", img.Name, "bad image name")
	assert.Equal(t, "latest", img.GetTag(), "bad image tag")
}

func TestImageIsSameKind(t *testing.T) {
	assert.True(t, New("rocker-build").IsSameKind(*New("rocker-build")))
	assert.True(t, New("rocker-build:latest").IsSameKind(*New("rocker-build:latest")))
	assert.True(t, New("rocker-build").IsSameKind(*New("rocker-build:1.2.1")))
	assert.True(t, New("rocker-build:1.2.1").IsSameKind(*New("rocker-build:1.2.1")))
	assert.True(t, New("grammarly/rocker-build").IsSameKind(*New("grammarly/rocker-build")))
	assert.True(t, New("grammarly/rocker-build:3.1").IsSameKind(*New("grammarly/rocker-build:3.1")))
	assert.True(t, New("grammarly/rocker-build").IsSameKind(*New("grammarly/rocker-build:3.1")))
	assert.True(t, New("grammarly/rocker-build:latest").IsSameKind(*New("grammarly/rocker-build:latest")))
	assert.True(t, New("quay.io/rocker-build").IsSameKind(*New("quay.io/rocker-build")))
	assert.True(t, New("quay.io/rocker-build:latest").IsSameKind(*New("quay.io/rocker-build:latest")))
	assert.True(t, New("quay.io/rocker-build:3.2").IsSameKind(*New("quay.io/rocker-build:3.2")))
	assert.True(t, New("quay.io/rocker-build").IsSameKind(*New("quay.io/rocker-build:3.2")))
	assert.True(t, New("quay.io/grammarly/rocker-build").IsSameKind(*New("quay.io/grammarly/rocker-build")))
	assert.True(t, New("quay.io/grammarly/rocker-build:latest").IsSameKind(*New("quay.io/grammarly/rocker-build:latest")))
	assert.True(t, New("quay.io/grammarly/rocker-build:1.2.1").IsSameKind(*New("quay.io/grammarly/rocker-build:1.2.1")))
	assert.True(t, New("quay.io/grammarly/rocker-build").IsSameKind(*New("quay.io/grammarly/rocker-build:1.2.1")))

	assert.False(t, New("rocker-build").IsSameKind(*New("rocker-build2")))
	assert.False(t, New("rocker-build").IsSameKind(*New("rocker-compose")))
	assert.False(t, New("rocker-build").IsSameKind(*New("grammarly/rocker-build")))
	assert.False(t, New("rocker-build").IsSameKind(*New("quay.io/rocker-build")))
	assert.False(t, New("rocker-build").IsSameKind(*New("quay.io/grammarly/rocker-build")))
}

func TestTagsGetOld(t *testing.T) {
	tags := Tags{
		Items: []*Tag{
			&Tag{"1", *New("hub/ns/name:1"), time.Unix(1, 0).Unix()},
			&Tag{"3", *New("hub/ns/name:3"), time.Unix(3, 0).Unix()},
			&Tag{"2", *New("hub/ns/name:2"), time.Unix(2, 0).Unix()},
			&Tag{"5", *New("hub/ns/name:5"), time.Unix(5, 0).Unix()},
			&Tag{"4", *New("hub/ns/name:4"), time.Unix(4, 0).Unix()},
		},
	}
	old := tags.GetOld(2)

	assert.Equal(t, 3, len(old), "bad number of old tags")
	assert.Equal(t, "hub/ns/name:3", old[0].String(), "bad old image 1")
	assert.Equal(t, "hub/ns/name:2", old[1].String(), "bad old image 2")
	assert.Equal(t, "hub/ns/name:1", old[2].String(), "bad old image 3")
}
