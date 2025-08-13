package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Pane struct {
	Name string `mapstructure:"name"` // shown in tmux title
	Dir  string `mapstructure:"dir"`
	Cmd  string `mapstructure:"cmd"`
}

type Layout struct {
	Columns []struct {
		Rows []string `mapstructure:"rows"` // pane keys in top→bottom order
	} `mapstructure:"columns"`
}

type Profile struct {
	Layout Layout          `mapstructure:"layout"`
	Panes  map[string]Pane `mapstructure:"panes"`
}

type Root struct {
	Version     int                `mapstructure:"version"`
	SessionName string             `mapstructure:"sessionName"`
	Profiles    map[string]Profile `mapstructure:"profiles"`
}

func LoadConfig() (*Root, error) {
	v := viper.New()

	// Project-level first
	v.SetConfigName(".devmod")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	err := v.ReadInConfig()

	if err != nil {
		// Fallback to global: ~/.config/devmod/config.yml
		home, _ := os.UserHomeDir()
		globalPath := filepath.Join(home, ".config", "devmod")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(globalPath)

		if err2 := v.ReadInConfig(); err2 != nil {
			return nil, fmt.Errorf("no config found: %w", err2)
		}
	}

	var cfg Root
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	if cfg.SessionName == "" {
		cfg.SessionName = "devmod"
	}
	return &cfg, nil
}
