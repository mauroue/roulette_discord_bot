package main

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Token         string `yaml:"token"`
	Target        string `yaml:"target"`
	TargetChannel string `yaml:"target-channel"`
	TargetServer  string `yaml:"target-server"`
	DatabasePath  string `yaml:"db-path"`
}

var cfg *Config

func LoadConfigFromFile(filename string) error {
	// Load config from specified file and parse using yaml decoder
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	decode := yaml.NewDecoder(file)

	if err := decode.Decode(&cfg); err != nil {
		return err
	}

	return err
}
