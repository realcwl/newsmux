package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateCoinbase(t *testing.T) {
	assert.True(t, ContainsString([]string{"a", "b"}, "a"))
	assert.False(t, ContainsString([]string{}, "a"))
	assert.False(t, ContainsString([]string{"a", "b"}, "c"))
}
