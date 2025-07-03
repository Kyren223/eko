package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/kyren223/eko/internal/client/ui/colors"
)

type Config struct {
	ServerName             string   `json:"server_name"`
	PrivateKeyPath         string   `json:"private_key_path"`
	InsertModeTabToSpace   bool     `json:"insert_mode_tab_to_space"`
	InsertModeSpacesPerTab uint8    `json:"insert_mode_spaces_per_tab"`
	InsecureDebugMode      bool     `json:"insecure_debug_mode"`
	Colors                 []string `json:"colors"`
}

func Default() Config {
	return Config{
		ServerName:             "eko.kyren.codes",
		PrivateKeyPath:         "",
		InsertModeTabToSpace:   true,
		InsertModeSpacesPerTab: 4,
		InsecureDebugMode:      false,
		Colors:                 nil,
	}
}

func VerifyAndFix(config *Config) error {
	if config.ServerName == "" {
		config.ServerName = Default().ServerName
	}

	if config.Colors != nil {
		for i, color := range config.Colors {
			if !colors.IsHex(color) {
				return fmt.Errorf("color at %v is not a valid hex color", i)
			}
		}

		if len(config.Colors) != colors.Count {
			return fmt.Errorf(
				"expected %v colors, got %v colors, set this value to null to use the default colors",
				colors.Count, len(config.Colors),
			)
		}

		colors.LoadStrings(config.Colors)
	}

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

	err = VerifyAndFix(&config)
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
