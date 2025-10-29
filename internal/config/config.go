package config

import (
	"encoding/json"
	"os"
)

// Create new config instance
func NewConfig() *Config {
	return &Config{}
}

// Load configuration file in json format
func (c *Config) Read(file string) error {
	data, err := os.ReadFile(file)
	if err == nil {
		_ = json.Unmarshal(data, c)
	}
	return err
}
