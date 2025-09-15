package config

import (
	"log"
	"os"
)

type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
}

type DatabaseConfig struct {
	Host     string
	User     string
	Password string
	Name     string
}

type ServerConfig struct {
	Port string
}

func LoadConfig() *Config {
	config := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			User:     getEnv("DB_USER", "APG"),
			Password: getEnv("DB_PASSWORD", "123ciao"),
			Name:     getEnv("DB_NAME", "ProgCarc"),
		},
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
	}

	// Verifica che le variabili critiche siano presenti
	if config.Database.Host == "" || config.Database.User == "" || 
	   config.Database.Password == "" || config.Database.Name == "" {
		log.Fatal("Database environment variables are not set")
	}

	return config
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func (c *Config) GetDSN() string {
	return "host=" + c.Database.Host + 
		   " user=" + c.Database.User + 
		   " password=" + c.Database.Password + 
		   " dbname=" + c.Database.Name + 
		   " sslmode=disable"
}