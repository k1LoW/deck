package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

var (
	dataHomePath  string
	stateHomePath string
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

// Load loads the configuration from the config file.
// It searches for config files in the following order:
// 1. $XDG_DATA_HOME/deck/config-{profile}.yml
// 2. $XDG_DATA_HOME/deck/config.yml
// If no config file is found, it returns an empty Config struct.
func Load(profile string) (*Config, error) {
	var configBasePaths []string
	if profile != "" {
		configBasePaths = append(configBasePaths, filepath.Join(DataHomePath(), fmt.Sprintf("config-%s", profile)))
	}
	configBasePaths = append(configBasePaths, filepath.Join(DataHomePath(), "config"))
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

// DataHomePath returns the path to the data home directory.
func DataHomePath() string {
	if dataHomePath != "" {
		return dataHomePath
	}
	if os.Getenv("XDG_DATA_HOME") != "" {
		dataHomePath = filepath.Join(os.Getenv("XDG_DATA_HOME"), "deck")
	}
	dataHomePath = filepath.Join(os.Getenv("HOME"), ".local", "share", "deck")
	return dataHomePath
}

func StateHomePath() string {
	if stateHomePath != "" {
		return stateHomePath
	}
	if os.Getenv("XDG_STATE_HOME") != "" {
		stateHomePath = filepath.Join(os.Getenv("XDG_STATE_HOME"), "deck")
	}
	stateHomePath = filepath.Join(os.Getenv("HOME"), ".local", "state", "deck")
	return stateHomePath
}
