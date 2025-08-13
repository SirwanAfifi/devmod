package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type ProjectConfig struct {
	SessionName string `mapstructure:"sessionName"`
	BackendDir  string `mapstructure:"backendDir"`
	BackendCmd  string `mapstructure:"backendCmd"`
	FrontendDir string `mapstructure:"frontendDir"`
	FrontendCmd string `mapstructure:"frontendCmd"`
	VcsDir      string `mapstructure:"vcsDir"`
}

// LoadConfig tries to load project config first, then global config
func LoadConfig() (*ProjectConfig, error) {
	v := viper.New()

	// 1. Try project config: ./.devmod.yml
	v.SetConfigName(".devmod")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	err := v.ReadInConfig()

	if err != nil {
		// 2. Fallback to global config: ~/.config/devmod/config.yml
		home, _ := os.UserHomeDir()
		globalPath := filepath.Join(home, ".config", "devmod")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(globalPath)

		if err2 := v.ReadInConfig(); err2 != nil {
			return nil, fmt.Errorf("no config found: %w", err2)
		}
	}

	var cfg ProjectConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return &cfg, nil
}
