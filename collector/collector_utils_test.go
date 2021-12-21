package collector

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConcateUrlBaseAndRelativePath(t *testing.T) {
	require.Equal(t, "a.com/b", ConcateUrlBaseAndRelativePath("a.com", "b"))
	require.Equal(t, "a.com/b", ConcateUrlBaseAndRelativePath("a.com", "/b"))
	require.Equal(t, "a.com/b", ConcateUrlBaseAndRelativePath("a.com/", "b"))
	require.Equal(t, "a.com/b", ConcateUrlBaseAndRelativePath("a.com/", "/b"))
	require.Equal(t, "a.com/b", ConcateUrlBaseAndRelativePath("a.com//", "//b"))
}
