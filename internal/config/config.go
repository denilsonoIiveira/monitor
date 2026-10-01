package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Token          string  `mapstructure:"token"`
	AuthID         string  `mapstructure:"auth_id"`
	Cookie         string  `mapstructure:"cookie"`
	SpendAlertUSD  float64 `mapstructure:"spend_alert_usd"`
	RefreshSec     int     `mapstructure:"refresh_sec"`
	TeamID         int64   `mapstructure:"team_id"`
}

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "cursor-monitor"), nil
}

func ConfigFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func LoadConfig() (*Config, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}

	v := viper.New()
	v.AddConfigPath(dir)
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	v.SetDefault("refresh_sec", 30)
	v.SetDefault("spend_alert_usd", 50.0)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := ConfigFile()
	if err != nil {
		return err
	}

	content := fmt.Sprintf("# Cursor Cost Monitor Configuration\ntoken: %q\nauth_id: %q\ncookie: %q\nspend_alert_usd: %.2f\nrefresh_sec: %d\nteam_id: %d\n",
		cfg.Token,
		cfg.AuthID,
		cfg.Cookie,
		cfg.SpendAlertUSD,
		cfg.RefreshSec,
		cfg.TeamID,
	)

	return os.WriteFile(file, []byte(content), 0600)
}
