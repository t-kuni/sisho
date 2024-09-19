package testUtil

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

type Space struct {
	t       *testing.T
	Dir     string
	CleanUp func()
}

func BeginTestSpace(t *testing.T) Space {
	t.Helper()

	originalDir, err := os.Getwd()
	assert.NoError(t, err)

	tempDir, err := os.MkdirTemp("", "")
	assert.NoError(t, err)

	os.Chdir(tempDir)

	cleanup := func() {
		os.Chdir(originalDir)
		os.RemoveAll(tempDir)
	}

	return Space{
		t:       t,
		Dir:     tempDir,
		CleanUp: cleanup,
	}
}

func (s Space) WriteFile(path string, content []byte) {
	s.t.Helper()

	dir := filepath.Dir(path)

	err := os.MkdirAll(dir, os.ModePerm)
	assert.NoError(s.t, err)

	err = os.WriteFile(path, content, 0644)
	assert.NoError(s.t, err)
}

func (s Space) AssertFile(path string, assertion func(actual []byte)) {
	s.t.Helper()

	actual, err := os.ReadFile(path)
	assert.NoError(s.t, err)

	assertion(actual)
}

func (s Space) AssertExistPath(path string) {
	s.t.Helper()

	_, err := os.Stat(path)
	assert.NoError(s.t, err)
}
