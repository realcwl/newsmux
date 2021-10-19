package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestGetRandomNumberInRangeStandardDeviation(t *testing.T) {
	num := GetRandomNumberInRangeStandardDeviation(1, 1)
	assert.True(t, num >= 0 && num <= 2)

	num = GetRandomNumberInRangeStandardDeviation(5, 2)
	assert.True(t, num >= 3 && num <= 7)
}

func TestStringSlicesContainSameElements(t *testing.T) {
	assert.True(t, StringSlicesContainSameElements([]string{}, []string{}))
	assert.True(t, StringSlicesContainSameElements([]string{"a", "b"}, []string{"a", "b"}))
	assert.True(t, StringSlicesContainSameElements([]string{"a", "b"}, []string{"b", "a"}))
	assert.False(t, StringSlicesContainSameElements([]string{"a", "b"}, []string{"b", "a", "c"}))
	assert.False(t, StringSlicesContainSameElements([]string{"a", "b"}, []string{"b", "a", "a"}))
}
func TestMd5Hash(t *testing.T) {
	res, err := TextToMd5Hash("123")
	assert.NoError(t, err)
	assert.Equal(t, "202cb962ac59075b964b07152d234b70", res)
}

func TestGetUrlExtNameWithDot(t *testing.T) {
	require.Equal(t, ".jpg", GetUrlExtNameWithDot("https://wx2.sinaimg.cn/orj360/001SZvN4gy1gvf09rkb6rj61kp0u0th602.jpg"))
}
