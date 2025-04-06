package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BasePath string   `yaml:"base_path"`
	Modules  []Module `yaml:"modules"`
}

type Module struct {
	Path      string   `yaml:"path"`
	DependsOn []string `yaml:"depends_on,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
