package config

import (
	"os"
	"path/filepath"
)

var (
	dataHomePath  string
	stateHomePath string
)

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
