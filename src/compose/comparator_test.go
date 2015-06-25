package compose

import (
	"testing"
	"github.com/stretchr/testify/assert"
)


func TestComparator(t *testing.T) {
	cmp := NewComparator()
	containers := make([]*Container,0)
	act, err := cmp.Compare(containers, containers)
	assert.Empty(t, act)
	assert.Nil(t, err)
}
