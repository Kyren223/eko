package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

type Config struct {
	PrivateKeyPath         string `json:"private_key_path"`
	InsertModeTabToSpace   bool   `json:"insert_mode_tab_to_space"`
	InsertModeSpacesPerTab uint8  `json:"insert_mode_spaces_per_tab"`
}

func Default() Config {
	return Config{
		PrivateKeyPath:         "",
		InsertModeTabToSpace:   true,
		InsertModeSpacesPerTab: 4,
	}
}

func Verify(config *Config) error {
	return nil
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
	err = os.MkdirAll(Dir, 0o750)
	if err != nil {
		return err
	}

	ConfigFile = filepath.Join(Dir, "config.json")
	contents, err := os.ReadFile(ConfigFile) // #nosec 304
	if errors.Is(err, os.ErrNotExist) {
		config = Default()
		return write()
	}
	if err != nil {
		return err
	}

	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(contents, &rawMap); err != nil {
		return err
	}

	defaultVal := reflect.ValueOf(Default())
	finalConfig := reflect.New(defaultVal.Type()).Elem()
	finalConfig.Set(defaultVal)

	for i := 0; i < defaultVal.NumField(); i++ {
		field := defaultVal.Type().Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = field.Name
		}

		rawValue, found := rawMap[jsonTag]
		fieldValue := finalConfig.Field(i)
		if !found || !fieldValue.CanAddr() {
			continue
		}

		err := json.Unmarshal(rawValue, fieldValue.Addr().Interface())
		if err != nil {
			return fmt.Errorf("error unmarshaling field %s: %w", field.Name, err)
		}
	}

	config = finalConfig.Interface().(Config)

	err = Verify(&config)
	if err != nil {
		return err
	}

	return write()
}

func write() error {
	b, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFile, b, 0o600)
}

func Read() Config {
	return config
}

func Use(f func(config *Config)) error {
	f(&config)
	return write()
}
