package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// AICoderConfig holds the configuration for the application.
type AICoderConfig struct {
	Repository string       `yaml:"repository"`
	Load       LoadConfig   `yaml:"load"`
	Search     SearchConfig `yaml:"search"`
}

type LoadConfig struct {
	Exclude []string `yaml:"exclude"`
	Include []string `yaml:"include"`
}

func (c *LoadConfig) IsExcluded(path string) bool {
	for _, excl := range c.Exclude {
		if matchesPath(path, excl) {
			return true
		}
	}
	return false
}

// isIncluded checks if a given path is explicitly included based on the include list.
func (c *LoadConfig) IsIncluded(path string) bool {
	for _, incl := range c.Include {
		if matchesPath(path, incl) {
			return true
		}
	}
	return false
}

func matchesPath(target, pattern string) bool {
	return strings.HasPrefix(target, pattern)
}

type SearchConfig struct {
	TopN int `yaml:"top_n"`
}

// cfg holds the loaded configuration.
var cfg AICoderConfig

// ReadConfig loads the configuration from the specified file.
func ReadConfig(cfgFile string) {
	absPath, err := filepath.Abs(cfgFile)
	if err != nil {
		log.Fatalf("Failed to get absolute path of config file: %v", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		log.Fatalf("Failed to read config file %s: %v", absPath, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	log.Printf("Configuration loaded from %s", absPath)
}

// GetConfig returns the loaded configuration.
func GetConfig() AICoderConfig {
	return cfg
}

func GetLoadConfig() LoadConfig {
	return cfg.Load
}

func GetSearchConfig() SearchConfig {
	return cfg.Search
}
