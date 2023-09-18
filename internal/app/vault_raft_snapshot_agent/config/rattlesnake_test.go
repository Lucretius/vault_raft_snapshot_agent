package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent/test"
)

type rattlesnakeConfigStub struct {
	Path string `default:"/test/file" resolve-path:""`
	Url	 string `validate:"omitempty,http_url"`
}

func TestUnmarshalResolvesRelativePaths(t *testing.T) {
	rattlesnake := newRattlesnake("test", "TEST")

	wd, err := os.Getwd()
	assert.NoError(t, err, "Getwd failed unexpectedly")

	err = rattlesnake.SetConfigFile(fmt.Sprintf("%s/config.yml", wd))
	assert.NoError(t, err, "SetConfigFile failed unexpectedly")

	t.Setenv("TEST_PATH", "./file.ext")
	config := rattlesnakeConfigStub{}
	err = rattlesnake.Unmarshal(&config)

	assert.NoError(t, err, "Unmarshal failed unexpectedly")
	assert.Equal(t, filepath.Clean(fmt.Sprintf("%s/file.ext", wd)), config.Path)
}

func TestUnmarshalSetsDefaultValues(t *testing.T) {
	rattlesnake := newRattlesnake("test", "TEST")

	config := rattlesnakeConfigStub{}
	err := rattlesnake.Unmarshal(&config)

	assert.NoError(t, err, "Unmarshal failed unexpectedly")
	assert.Equal(t, "/test/file", config.Path)
}

func TestUnmarshalValidatesValues(t *testing.T) {
	rattlesnake := newRattlesnake("test", "TEST")

	t.Setenv("TEST_URL", "not_an_url")
	config := rattlesnakeConfigStub{}
	err := rattlesnake.Unmarshal(&config)

	assert.Error(t, err, "Unmarshal should fail on validation error")
	assert.Equal(t, "not_an_url", config.Url)
}

func TestOnConfigChangeRunsHandler(t *testing.T) {
	rattlesnake := newRattlesnake("test", "TEST")
	configFile := fmt.Sprintf("%s/config.yml", t.TempDir())

	err := rattlesnake.SetConfigFile(configFile)
	assert.NoError(t, err, "SetConfigFile failed unexpectedly")

	err = test.WriteFile(t, configFile, "{\"url\": \"http://example.com\"}")
	assert.NoError(t, err, "writing config file failed unexpectedly")

	err = rattlesnake.Unmarshal(&rattlesnakeConfigStub{})
	assert.NoError(t, err, "Unmarshal failed unexpectedly")

	changed := make(chan bool, 1)
	rattlesnake.OnConfigChange(func() {
		changed <- true
	})

	err = test.WriteFile(t, configFile, "{\"url\": \"http://new.com\"}")
	assert.NoError(t, err, "writing config file failed unexpectedly")

	assert.True(t, <-changed)
}
