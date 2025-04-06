package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Environment string   `yaml:"environment"`
	Modules     []Module `yaml:"modules"`
}

type Module struct {
	Name    string `yaml:"name"`
	Service string `yaml:"service"`
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
