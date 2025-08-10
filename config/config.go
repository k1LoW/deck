package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

var (
	homePath       string
	configHomePath string
	dataHomePath   string
	stateHomePath  string
)

type Config struct {
	// Whether to display line breaks in the document as line breaks
	Breaks *bool `yaml:"breaks,omitempty" json:"breaks,omitempty"`
	// Conditions for default
	Defaults []DefaultCondition `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	// command to convert code blocks to images
	CodeBlockToImageCommand string `yaml:"codeBlockToImageCommand,omitempty" json:"codeBlockToImageCommand,omitempty"`
}

type DefaultCondition struct {
	If     string `json:"if"`               // condition to check
	Layout string `json:"layout,omitempty"` // layout name to apply if condition is true
	Freeze *bool  `json:"freeze,omitempty"` // freeze the page
	Ignore *bool  `json:"ignore,omitempty"` // whether to ignore the page if condition is true
	Skip   *bool  `json:"skip,omitempty"`   // whether to skip the page if condition is true
}

func init() {
	var err error
	homePath, err = os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get home directory: %v", err))
	}
}

// Load loads the configuration from the config file.
// It searches for config files in the following order:
// 1. $XDG_CONFIG_HOME/deck/config-{profile}.yml
// 2. $XDG_CONFIG_HOME/deck/config.yml
// If no config file is found, it returns an empty Config struct.
func Load(profile string) (*Config, error) {
	var configBasePaths []string
	if profile != "" {
		configBasePaths = append(configBasePaths, filepath.Join(configPath(), fmt.Sprintf("config-%s", profile)))
	}
	configBasePaths = append(configBasePaths, filepath.Join(configPath(), "config"))
	cfg := &Config{}
	for _, basePath := range configBasePaths {
		for _, ext := range []string{".yml", ".yaml"} {
			configPath := basePath + ext
			if b, err := os.ReadFile(configPath); err == nil {
				if err := yaml.Unmarshal(b, cfg); err != nil {
					return nil, fmt.Errorf("failed to unmarshal config: %w", err)
				}
				return cfg, nil
			}
		}
	}
	// If no config file is found, return an empty config
	return cfg, nil
}

// configPath returns the path to the configuration directory.
func configPath() string {
	if configHomePath != "" {
		return configHomePath
	}
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		configHomePath = filepath.Join(v, "deck")
	} else {
		configHomePath = filepath.Join(homePath, ".config", "deck")
	}
	return configHomePath
}

// DataHomePath returns the path to the data home directory.
func DataHomePath() string {
	if dataHomePath != "" {
		return dataHomePath
	}
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		dataHomePath = filepath.Join(v, "deck")
	} else {
		dataHomePath = filepath.Join(homePath, ".local", "share", "deck")
	}
	return dataHomePath
}

func StateHomePath() string {
	if stateHomePath != "" {
		return stateHomePath
	}
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		stateHomePath = filepath.Join(v, "deck")
	} else {
		stateHomePath = filepath.Join(homePath, ".local", "state", "deck")
	}
	return stateHomePath
}
