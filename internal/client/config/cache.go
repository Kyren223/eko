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
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/kyren223/eko/pkg/assert"
)

type Cache struct {
	// Key is a server URL like "eko.kyren.codes" and value is the base 16 hash
	TosHashes map[string]string `json:"tos_hashes"`
	DeviceID  string            `json:"device_id"`
}

func DefaultCache() Cache {
	return Cache{
		TosHashes: map[string]string{},
		DeviceID:  "",
	}
}

func VerifyAndFixCache(cache *Cache) error {
	if cache.DeviceID == "" {
		cache.DeviceID = GenerateDeviceID()
	}
	return nil
}

func GenerateDeviceID() string {
	// Generate a random 32-byte device ID
	// Gurantees uniqueness, so it's not reverseable and it is annonymized
	// For legal reasons: this may still be logged on the server, so associated with an IP address
	deviceId := [32]byte{}
	_, err := rand.Read(deviceId[:])
	assert.NoError(err, "random never fails")
	return fmt.Sprintf("%x", deviceId)
}

var (
	cache     = Cache{}
	CacheDir  string
	CacheFile string
)

func LoadCache() error {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	CacheDir = filepath.Join(userCacheDir, "eko")
	err = os.MkdirAll(CacheDir, 0o750)
	if err != nil {
		return err
	}

	CacheFile = filepath.Join(CacheDir, "cache.json")
	contents, err := os.ReadFile(CacheFile) // #nosec 304
	if errors.Is(err, os.ErrNotExist) {
		cache = DefaultCache()
		return writeCache()
	}
	if err != nil {
		return err
	}

	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(contents, &rawMap); err != nil {
		return err
	}

	defaultVal := reflect.ValueOf(DefaultCache())
	finalCache := reflect.New(defaultVal.Type()).Elem()
	finalCache.Set(defaultVal)

	for i := 0; i < defaultVal.NumField(); i++ {
		field := defaultVal.Type().Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = field.Name
		}

		rawValue, found := rawMap[jsonTag]
		fieldValue := finalCache.Field(i)
		if !found || !fieldValue.CanAddr() {
			continue
		}

		err := json.Unmarshal(rawValue, fieldValue.Addr().Interface())
		if err != nil {
			return fmt.Errorf("error unmarshaling field %s: %w", field.Name, err)
		}
	}

	cache = finalCache.Interface().(Cache)

	err = VerifyAndFixCache(&cache)
	if err != nil {
		return err
	}

	return writeCache()
}

func writeCache() error {
	b, err := json.MarshalIndent(cache, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(CacheFile, b, 0o600)
}

func ReadCache() Cache {
	return cache
}

func UseCache(f func(cache *Cache)) error {
	f(&cache)
	return writeCache()
}
