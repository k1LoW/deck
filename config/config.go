package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/goccy/go-yaml"
)

type Config struct {
	// Whether to display line breaks in the document as line breaks
	Breaks *bool `yaml:"breaks,omitempty" json:"breaks,omitempty"`
	// Conditions for default
	Defaults []DefaultCondition `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	// command to convert code blocks to images
	CodeBlockToImageCommand string `yaml:"codeBlockToImageCommand,omitempty" json:"codeBlockToImageCommand,omitempty"`
	// folder ID to create presentations and upload temporary images to
	FolderID string `yaml:"folderID,omitempty" json:"folderID,omitempty"`
}

type DefaultCondition struct {
	If     string `json:"if"`               // condition to check
	Layout string `json:"layout,omitempty"` // layout name to apply if condition is true
	Freeze *bool  `json:"freeze,omitempty"` // freeze the page
	Ignore *bool  `json:"ignore,omitempty"` // whether to ignore the page if condition is true
	Skip   *bool  `json:"skip,omitempty"`   // whether to skip the page if condition is true
}

var homeDir string

func init() {
	var err error
	homeDir, err = os.UserHomeDir()
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
		configBasePaths = append(configBasePaths, filepath.Join(configHomePath(), fmt.Sprintf("config-%s", profile)))
	}
	configBasePaths = append(configBasePaths, filepath.Join(configHomePath(), "config"))
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

// On macOS, we use directories that conform to the XDG Base Directory instead of `os.UserConfigDir`
// or `os.UserDataDir`, etc. It is more intuitive for CLI applications.

var configHomePath = sync.OnceValue(func() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "deck")
	}
	return filepath.Join(homeDir, ".config", "deck")
})

var dataHomePath = sync.OnceValue(func() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "deck")
	}
	return filepath.Join(homeDir, ".local", "share", "deck")
})

var stateHomePath = sync.OnceValue(func() string {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return filepath.Join(v, "deck")
	}
	return filepath.Join(homeDir, ".local", "state", "deck")
})

// DataHomePath returns the path to the data home directory.
func DataHomePath() string {
	return dataHomePath()
}

func StateHomePath() string {
	return stateHomePath()
}
