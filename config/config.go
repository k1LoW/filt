package config

import (
	"errors"
	"html/template"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// avairable config and default
var configs = map[string]interface{}{
	"history.enable": false,
	"history.path":   filepath.Join(dataPath(), "history"),
}

// Load ...
func Load() {
	for k, v := range configs {
		viper.SetDefault(k, v)
	}
	viper.SetConfigType("toml")
	viper.SetConfigName("config")
	viper.AddConfigPath(configPath())
	_ = viper.ReadInConfig()
}

// Set value to config
func Set(k, v string) error {
	if !isExist(k) {
		return errors.New("invalid config key")
	}
	viper.Set(k, v)
	return nil
}

// Save config.toml
func Save() error {
	const cfgTemplate = `[history]
enable = {{ .history.enable }}
path = "{{ .history.path }}"
`
	tpl, err := template.New("config").Parse(cfgTemplate)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configPath(), os.ModeDir|0700); err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(configPath(), "config.toml"), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	if err := tpl.Execute(file, viper.AllSettings()); err != nil {
		return err
	}
	return nil
}

func isExist(k string) bool {
	_, ok := configs[k]
	return ok
}

func configPath() string {
	p := os.Getenv("XDG_CONFIG_HOME")
	if p == "" {
		home := os.Getenv("HOME")
		p = filepath.Join(home, ".config")
	}
	return filepath.Join(p, "filt")
}

func dataPath() string {
	p := os.Getenv("XDG_DATA_HOME")
	if p == "" {
		home := os.Getenv("HOME")
		p = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(p, "filt")
}
