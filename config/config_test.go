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
contexts:
  default:
    target_path: cmd
    exclude:
      - ent
      - go.sum
      - repo_structure.json
    include:
      - ent/schema
current_context: default
search:
  top_n: 10
`

	// Initialize the config
	initConfig(bytes.NewReader([]byte(configContent)))

	// Get the config
	config := GetConfig()

	// Assert the values
	assert.Equal(t, "aicoder", config.Repository)
	assert.Equal(t, "default", config.CurrentContext)
	assert.Equal(t, 10, config.Search.TopN)

	// Test the current context's LoadConfig
	currentLoadConfig := config.GetCurrentLoadConfig()
	assert.Equal(t, "cmd", currentLoadConfig.TargetPath)
	assert.Contains(t, currentLoadConfig.Exclude, "ent")
	assert.Contains(t, currentLoadConfig.Include, "ent/schema")

	searchConfig := GetSearchConfig()
	assert.Equal(t, 10, searchConfig.TopN)
}

// TestDefaultConfig tests the default configuration
func TestDefaultConfig(t *testing.T) {
	// mock getWorkingDirectory
	getWorkingDirectory = func() (string, error) {
		return "/tmp/aicoder", nil
	}
	defaultConfig, err := getDefaultConfig()
	assert.Nil(t, err)

	// Check if the default config contains the repository name
	assert.Contains(t, string(defaultConfig), "repository: aicoder")
	assert.Contains(t, string(defaultConfig), "current_context: default")
}
