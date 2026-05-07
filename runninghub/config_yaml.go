package runninghub

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

type configPathSetter interface {
	SetConfigPath(path string)
}

type yamlDefaultApplier interface {
	ApplyYAMLDefaults()
}

// LoadYAMLConfig loads a YAML config file into a typed struct.
// If the target config type implements ApplyYAMLDefaults, defaults are applied
// before unmarshalling and before writing a missing file.
func LoadYAMLConfig[T any](configFile string) (*T, error) {
	if configFile == "" {
		return nil, errors.New("config file path cannot be empty")
	}

	config := new(T)
	if applier, ok := any(config).(yamlDefaultApplier); ok {
		applier.ApplyYAMLDefaults()
	}

	if _, err := os.Stat(configFile); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}

		if setter, ok := any(config).(configPathSetter); ok {
			setter.SetConfigPath(configFile)
		}
		if err := SaveYAMLConfig(configFile, config); err != nil {
			return nil, err
		}
		return config, nil
	}

	raw, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(raw, config); err != nil {
		return nil, err
	}
	if setter, ok := any(config).(configPathSetter); ok {
		setter.SetConfigPath(configFile)
	}
	return config, nil
}

// SaveYAMLConfig writes a config struct to a YAML file.
func SaveYAMLConfig(configFile string, config any) error {
	if configFile == "" {
		return errors.New("config file path cannot be empty")
	}
	if config == nil {
		return errors.New("config cannot be nil")
	}

	if err := os.MkdirAll(filepath.Dir(configFile), 0o755); err != nil {
		return err
	}

	raw, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, raw, 0o644)
}