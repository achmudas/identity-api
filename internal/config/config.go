package config

import "github.com/caarlos0/env/v11"

type Config struct {
	DBUrl string `env:"DB_HOST"`
	Port  string `env:"APP_PORT"`
}

func Load() (Config, error) {
	cfg := Config{}
	if err := env.ParseWithOptions(&cfg, env.Options{
		RequiredIfNoDef: true,
	}); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
