package config

import (
	"github.com/caarlos0/env/v11"
	// autoload is needed to be imported only that's why it's blank
	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	DBConfig       DBConfig
	AppConfig      AppConfig
	KeycloakConfig KeycloakConfig
}

type DBConfig struct {
	DBUrl      string `env:"DB_HOST"`
	DBUsername string `env:"DB_USERNAME"`
	DBPassword string `env:"DB_PASSWORD"`
	DBSchema   string `env:"DB_SCHEMA"`
}

type AppConfig struct {
	Port       string `env:"APP_PORT"`
	ProfileUrl string `env:"PROFILE_URL"`
}

type KeycloakConfig struct {
	KeycloakURL          string `env:"KEYCLOAK_URL"`
	KeycloakRealm        string `env:"KEYCLOAK_REALM"`
	KeycloakClientID     string `env:"KEYCLOAK_CLIENT_ID"`
	KeycloakClientSecret string `env:"KEYCLOAK_CLIENT_SECRET"`
	KeycloakRedirectURL  string `env:"KEYCLOAK_REDIRECT_URL"`
}

func Load() (Config, error) {
	config := Config{}
	if err := env.ParseWithOptions(&config, env.Options{
		RequiredIfNoDef: true,
	}); err != nil {
		return Config{}, err
	}
	return config, nil
}
