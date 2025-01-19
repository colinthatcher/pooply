package main

import (
	"os"
)

var AppConfig *Config

type Config struct {
	AppID     string
	AuthToken string
	Postgres  PostgresConfig
}

type PostgresConfig struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
}

func init() {
	AppConfig = &Config{}
	AppConfig.AppID = getOrDefaultEnvVariable("APP_ID", "<app_id>")
	AppConfig.AuthToken = getOrDefaultEnvVariable("AUTH_TOKEN", "<auth_token>")
	AppConfig.Postgres.Host = getOrDefaultEnvVariable("POSTGRES_HOST", "localhost")
	AppConfig.Postgres.Port = getOrDefaultEnvVariable("POSTGRES_PORT", "5432")
	AppConfig.Postgres.Database = getOrDefaultEnvVariable("POSTGRES_DB", "postgres")
	AppConfig.Postgres.User = getOrDefaultEnvVariable("POSTGRES_USER", "postgres")
	AppConfig.Postgres.Password = getOrDefaultEnvVariable("POSTGRES_PASSWORD", "")
}

func getOrDefaultEnvVariable(name string, valueDefault string) string {
	if val, exists := os.LookupEnv(name); exists {
		return val
	} else {
		return valueDefault
	}
}
