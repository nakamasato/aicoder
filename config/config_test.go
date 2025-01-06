// config/config_test.go
package config

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	// Set up a temporary config file
	configContent := `
repository: aicoder
load:
  target_path: cmd
  exclude:
    - ent
    - go.sum
    - repo_structure.json
  include:
    - ent/schema
search:
  top_n: 10
`

	// Initialize the config
	InitConfig(bytes.NewReader([]byte(configContent)))

	// Assert the values
	assert.Equal(t, "aicoder", cfg.Repository)
	assert.Equal(t, "cmd", cfg.Load.TargetPath)
	assert.Equal(t, 10, cfg.Search.TopN)

	config := GetConfig()

	assert.Equal(t, "aicoder", config.Repository)
	assert.Equal(t, "cmd", config.Load.TargetPath)
	assert.Equal(t, 10, config.Search.TopN)

	loadConfig := GetLoadConfig()

	assert.Equal(t, "cmd", loadConfig.TargetPath)

	searchConfig := GetSearchConfig()

	assert.Equal(t, 10, searchConfig.TopN)
}
