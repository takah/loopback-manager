package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	BaseDir string  `mapstructure:"base_dir"`
	IPRange IPRange `mapstructure:"ip_range"`
}

type IPRange struct {
	Base  string `mapstructure:"base"`
	Start int    `mapstructure:"start"`
	End   int    `mapstructure:"end"`
}

func Load() *Config {
	cfg := &Config{
		BaseDir: expandPath("~/github"),
		IPRange: IPRange{
			Base:  "127.0.0",
			Start: 10,
			End:   254,
		},
	}

	if baseDir := os.Getenv("GITHUB_BASE_DIR"); baseDir != "" {
		cfg.BaseDir = expandPath(baseDir)
	}

	if viper.IsSet("base_dir") {
		cfg.BaseDir = expandPath(viper.GetString("base_dir"))
	}
	if viper.IsSet("ip_range.base") {
		cfg.IPRange.Base = viper.GetString("ip_range.base")
	}
	if viper.IsSet("ip_range.start") {
		cfg.IPRange.Start = viper.GetInt("ip_range.start")
	}
	if viper.IsSet("ip_range.end") {
		cfg.IPRange.End = viper.GetInt("ip_range.end")
	}

	return cfg
}

func expandPath(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}
	return path
}