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

func TestMin(t *testing.T) {
	assert.Equal(t, 1, Min(1, 2))
	assert.Equal(t, 1, Min(2, 1))
	assert.Equal(t, -1, Min(-1, 2))
	assert.Equal(t, 0, Min(0, 0))
}

func TestRandomAlphabetString(t *testing.T) {
	for i := 0; i < 100; i++ {
		first := RandomAlphabetString(8)
		second := RandomAlphabetString(8)
		assert.NotEqual(t, first, second)
	}
}

func TestStringSlicesContainSameElements(t *testing.T) {
	assert.True(t, StringSlicesContainSameElements([]string{}, []string{}))
	assert.True(t, StringSlicesContainSameElements([]string{"a", "b"}, []string{"a", "b"}))
	assert.True(t, StringSlicesContainSameElements([]string{"a", "b"}, []string{"b", "a"}))
	assert.False(t, StringSlicesContainSameElements([]string{"a", "b"}, []string{"b", "a", "c"}))
	assert.False(t, StringSlicesContainSameElements([]string{"a", "b"}, []string{"b", "a", "a"}))
}
