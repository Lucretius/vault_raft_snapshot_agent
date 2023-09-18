package config

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolvesRelativePaths(t *testing.T) {
	var test struct {
		Path         string `resolve-path:""`
		FixedPath    string `resolve-path:"/tmp/"`
		Other        string
		AbsolutePath string `resolve-path:""`
	}
	test.Path = "./relative"
	test.FixedPath = "./fixed"
	test.Other = "./other"
	test.AbsolutePath = "/test/abs"

	dir := t.TempDir()
	resolver := newPathResolver(dir)
	err := resolver.Resolve(&test)

	assert.NoError(t, err, "resolver.resolve failed unexepectedly")

	assert.Equal(t, filepath.Clean(fmt.Sprintf("%s/relative", dir)), test.Path)
	assert.Equal(t, filepath.Clean("/tmp/fixed"), test.FixedPath)
	assert.Equal(t, "/test/abs", test.AbsolutePath)
	assert.Equal(t, "./other", test.Other)
}

func TestResolvesRecursively(t *testing.T) {
	type inner struct {
		Path string `resolve-path:""`
	}

	innerPtr := inner{"./innerPtr"}

	var outer struct {
		Inner    inner
		InnerPtr *inner
	}
	outer.Inner.Path = "./inner"
	outer.InnerPtr = &innerPtr

	dir := t.TempDir()
	resolver := newPathResolver(dir)
	err := resolver.Resolve(&outer)

	assert.NoError(t, err, "resolver.resolve failed unexepectedly")

	assert.Equal(t, filepath.Clean(fmt.Sprintf("%s/inner", dir)), outer.Inner.Path)
	assert.Equal(t, filepath.Clean(fmt.Sprintf("%s/innerPtr", dir)), innerPtr.Path)

}
