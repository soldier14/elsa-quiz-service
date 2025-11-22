package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
		TTL      string `yaml:"ttl"`
	} `yaml:"redis"`
	Postgres struct {
		URL string `yaml:"url"`
	} `yaml:"postgres"`
	Quiz struct {
		TTL string `yaml:"ttl"`
	} `yaml:"quiz"`
}

// Load reads YAML config from path.
func Load(path string) (Config, error) {
	cfg := Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// TTLDuration parses a duration string or returns the fallback if empty.
func TTLDuration(raw string, fallback time.Duration) time.Duration {
	if raw == "" {
		return fallback
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d
	}
	return fallback
}
