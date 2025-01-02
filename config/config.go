package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// AICoderConfig holds the configuration for the application.
type AICoderConfig struct {
	Repository   string
	Load         LoadConfig
	Search       SearchConfig
	OpenAIAPIKey string
}

type LoadConfig struct {
	TargetPath string   `yaml:"target_path"` // Target path to load files from
	Exclude    []string `yaml:"exclude"`     // List of paths to exclude
	Include    []string `yaml:"include"`     // List of paths to include in excluded paths
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

// LoadConfig initializes the configuration using Viper
func InitConfig() {
	viper.SetConfigName(".aicoder") // Name of config file (without extension)
	viper.SetConfigType("yaml")     // Required if the config file does not have the extension in the name
	viper.AddConfigPath(".")        // Optionally look for config in the working directory

	// Set default values
	viper.SetDefault("repository", "default-repo")
	viper.SetDefault("load", LoadConfig{})
	viper.SetDefault("search", SearchConfig{})

	// Bind environment variables
	viper.BindEnv("openai_api_key", "OPENAI_API_KEY")

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Error reading config file, %s", err)
	}

	// Unmarshal the config into the struct
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Manually set the OpenAI API Key from environment variable
	cfg.OpenAIAPIKey = viper.GetString("openai_api_key")
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
