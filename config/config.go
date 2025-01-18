package config

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// AICoderConfig holds the configuration for the application.
type AICoderConfig struct {
	Repository     string                `mapstructure:"repository"`
	Contexts       map[string]LoadConfig `mapstructure:"contexts"`        // Contexts for different LoadConfigs
	CurrentContext string                `mapstructure:"current_context"` // Current context to use
	Search         SearchConfig          `mapstructure:"search"`
	OpenAIAPIKey   string                `mapstructure:"openai_api_key"`
}

// GetCurrentLoadConfig returns the LoadConfig for the current context.
func (c *AICoderConfig) GetCurrentLoadConfig() LoadConfig {
	if loadConfig, exists := c.Contexts[c.CurrentContext]; exists {
		return loadConfig
	}
	log.Fatalf("Current context '%s' not found in contexts %v", c.CurrentContext, c.Contexts)
	return LoadConfig{} // This line will never be reached due to log.Fatalf
}

type LoadConfig struct {
	TargetPath string   `mapstructure:"target_path"` // Target path to load files from
	Exclude    []string `mapstructure:"exclude"`     // List of paths to exclude
	Include    []string `mapstructure:"include"`     // List of paths to include in excluded paths
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
	TopN int `mapstructure:"top_n"`
}

// cfg holds the loaded configuration.
var cfg AICoderConfig

// LoadConfig initializes the configuration using Viper
func InitConfig(configFile string) {
	_, err := os.Stat(configFile)
	if err == nil {
		// if file exists
		fmt.Printf("loading config from %s\n", configFile)
		configReader, err := os.Open(filepath.Clean(configFile))
		if err != nil {
			log.Fatalf("failed to open config file: %v", err)
		}
		defer configReader.Close()
		initConfig(configReader)
	} else if os.IsNotExist(err) {
		// default bytestream if the file does not exist
		defaultConfig, err := getDefaultConfig()
		if err != nil {
			log.Fatalf("failed to get default config: %v", err)
		}
		configReader := bytes.NewReader(defaultConfig)
		initConfig(configReader)
	} else {
		log.Fatalf("failed to check if config file exists: %v", err)
	}
}

func getDefaultConfig() ([]byte, error) {
	// Get the current working directory
	cwd, err := getWorkingDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to get current dir: %v", err)
	}

	// Extract the directory name
	dirName := filepath.Base(cwd)
	log.Println(dirName)

	content := fmt.Sprintf(`repository: %s
contexts:
  default:
    target_path:
    exclude: []
    include: []
current_context: default
search:
  top_n: 5
`, dirName) // Default content for the .aicoder.yaml file
	return []byte(content), nil
}

func initConfig(reader io.Reader) {
	viper.SetConfigType("yaml")
	// Set default values
	// viper.SetDefault("repository", "default-repo")
	// viper.SetDefault("contexts.load.", LoadConfig{})
	// viper.SetDefault("search", SearchConfig{})

	// Bind environment variables
	if err := viper.BindEnv("openai_api_key", "OPENAI_API_KEY"); err != nil {
		log.Fatalf("Failed to bind environment variable: %v", err)
	}

	// Read the config file
	if err := viper.ReadConfig(reader); err != nil {
		fmt.Printf("Error reading config from reader, %s", err)
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
	return cfg.GetCurrentLoadConfig()
}

func GetSearchConfig() SearchConfig {
	return cfg.Search
}

func CreateDefaultConfigFile(writer io.Writer) error {

	content, err := getDefaultConfig()
	if err != nil {
		return fmt.Errorf("failed to get default config: %v", err)
	}

	_, err = writer.Write([]byte(content))
	return err
}

// getWorkingDirectory is a wrapper around os.Getwd
var getWorkingDirectory = func() (string, error) {
	return os.Getwd()
}
