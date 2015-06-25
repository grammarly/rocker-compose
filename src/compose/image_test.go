package compose

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewImageNameFromString_NoRegistry(t *testing.T) {
	str := "grammarly/rocker"
	image := NewImageNameFromString(str)
	expected := ImageName{"", "grammarly/rocker", "latest"}
	assert.Equal(t, expected, *image)
	assert.Equal(t, str+":latest", image.String())
}

func TestNewImageNameFromString_Library(t *testing.T) {
	str := "ubuntu"
	image := NewImageNameFromString(str)
	expected := ImageName{"", "ubuntu", "latest"}
	assert.Equal(t, expected, *image)
	assert.Equal(t, str+":latest", image.String())
}

func TestNewImageNameFromString_WithVersion(t *testing.T) {
	str := "grammarly/rocker:1.2.3"
	image := NewImageNameFromString(str)
	expected := ImageName{"", "grammarly/rocker", "1.2.3"}
	assert.Equal(t, expected, *image)
	assert.Equal(t, str, image.String())
}

func TestNewImageNameFromString_LibraryWithVersion(t *testing.T) {
	str := "ubuntu:12.04"
	image := NewImageNameFromString(str)
	expected := ImageName{"", "ubuntu", "12.04"}
	assert.Equal(t, expected, *image)
	assert.Equal(t, str, image.String())
}

func TestNewImageNameFromString_Registry(t *testing.T) {
	str := "quay.io/grammarly/rocker"
	image := NewImageNameFromString(str)
	expected := ImageName{"quay.io", "grammarly/rocker", "latest"}
	assert.Equal(t, expected, *image)
	assert.Equal(t, str+":latest", image.String())
}

func TestNewImageNameFromString_RegistryVersion(t *testing.T) {
	str := "quay.io/grammarly/rocker:1.2.3"
	image := NewImageNameFromString(str)
	expected := ImageName{"quay.io", "grammarly/rocker", "1.2.3"}
	assert.Equal(t, expected, *image)
	assert.Equal(t, str, image.String())
}

func TestNewImageNameFromString_RegistryLibrary(t *testing.T) {
	str := "quay.io/rocker"
	image := NewImageNameFromString(str)
	expected := ImageName{"quay.io", "rocker", "latest"}
	assert.Equal(t, expected, *image)
	assert.Equal(t, str+":latest", image.String())
}

func TestNewImageNameFromString_RegistryLibraryVersion(t *testing.T) {
	str := "quay.io/rocker:1.2.3"
	image := NewImageNameFromString(str)
	expected := ImageName{"quay.io", "rocker", "1.2.3"}
	assert.Equal(t, expected, *image)
	assert.Equal(t, str, image.String())
}
