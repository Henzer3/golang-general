package config

import (
	"flag"

	"github.com/ilyakaznacheev/cleanenv"
)

const configFile = "config.yaml"

type Config struct {
	Port string `env:"HELLO_PORT" env-default:"8080" yaml:"port"`
}

func Load() (*Config, error) {
	var cfg Config

	configPath := flag.String("config", configFile, "path for config file")
	flag.Parse()
	if err := cleanenv.ReadConfig(*configPath, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
