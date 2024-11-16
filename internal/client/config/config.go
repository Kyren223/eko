package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	PrivateKeyPath string
}

func Default() Config {
	return Config{
		PrivateKeyPath: "",
	}
}

var (
	config     = Config{}
	Dir        string
	ConfigFile string
)

func Load() error {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	Dir = filepath.Join(userConfigDir, "eko")
	err = os.MkdirAll(Dir, 0o755)
	if err != nil {
		return err
	}

	ConfigFile = filepath.Join(Dir, "config.json")
	contents, err := os.ReadFile(ConfigFile)
	if errors.Is(err, os.ErrNotExist) {
		config = Default()
		return write()
	}
	if err != nil {
		return err
	}

	err = json.Unmarshal(contents, &config)
	if err != nil {
		return err
	}

	return nil
}

func write() error {
	b, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFile, b, 0o644)
}

func Read() Config {
	return config
}

func Use(f func(config *Config)) error {
	f(&config)
	return write()
}
