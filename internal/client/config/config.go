// Eko: A terminal based social media platform
// Copyright (C) 2025 Kyren223
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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
	ServerName               string   `json:"server_name"`
	PrivateKeyPath           string   `json:"private_key_path"`
	InsertModeTabToSpace     bool     `json:"insert_mode_tab_to_space"`
	InsertModeSpacesPerTab   uint8    `json:"insert_mode_spaces_per_tab"`
	InsecureDebugMode        bool     `json:"insecure_debug_mode"`
	Colors                   []string `json:"colors"`
	AnonymousDeviceAnalytics bool     `json:"anonymous_device_analytics"`
	ScreenBorders            bool     `json:"screen_borders"`
}

func DefaultConfig() Config {
	return Config{
		ServerName:               "eko.kyren.codes",
		PrivateKeyPath:           "",
		InsertModeTabToSpace:     true,
		InsertModeSpacesPerTab:   4,
		InsecureDebugMode:        false,
		Colors:                   nil,
		AnonymousDeviceAnalytics: true,
		ScreenBorders:            true,
	}
}

func VerifyAndFixConfig(config *Config) error {
	if config.ServerName == "" {
		config.ServerName = DefaultConfig().ServerName
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
	ConfigDir  string
	ConfigFile string
)

func LoadConfig() error {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	ConfigDir = filepath.Join(userConfigDir, "eko")
	err = os.MkdirAll(ConfigDir, 0o750)
	if err != nil {
		return err
	}

	ConfigFile = filepath.Join(ConfigDir, "config.json")
	contents, err := os.ReadFile(ConfigFile) // #nosec 304
	if errors.Is(err, os.ErrNotExist) {
		config = DefaultConfig()
		return writeConfig()
	}
	if err != nil {
		return err
	}

	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(contents, &rawMap); err != nil {
		return err
	}

	defaultVal := reflect.ValueOf(DefaultConfig())
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

	err = VerifyAndFixConfig(&config)
	if err != nil {
		return err
	}

	return writeConfig()
}

func writeConfig() error {
	b, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFile, b, 0o600)
}

func ReadConfig() Config {
	return config
}

func UseConfig(f func(config *Config)) error {
	f(&config)
	return writeConfig()
}
