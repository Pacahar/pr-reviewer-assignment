package config

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Environment string     `yaml:"environment" env-required:"true"` // local, dev, production
	HTTPServer  HTTPServer `yaml:"http_server"`
	Database    DB         `yaml:"database"`
}

type HTTPServer struct {
	Address string `yaml:"address" env-default:"0.0.0.0"`
	Port    int    `yaml:"port" env-default:"4000"`
}

type DB struct {
	Host     string `yaml:"host" env-required:"true"`
	Port     int    `yaml:"port" env-required:"true"`
	Username string `yaml:"username" env-required:"true"`
	Password string `yaml:"password" env-required:"true"`
	DBName   string `yaml:"db_name" env-default:"assignment"`
}

func (db DB) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		db.Host, db.Port, db.Username, db.Password, db.DBName)
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")

	if configPath == "" {
		configPathPTR := flag.String("config", "./../../config/config.yaml", "path/to/config.yaml with configuration")
		flag.Parse()
		configPath = *configPathPTR
	}

	_, err := os.Stat(configPath)
	if err != nil {
		log.Fatalf("invalid or not set configPath: %s", configPath)
	}

	var config Config
	err = cleanenv.ReadConfig(configPath, &config)

	if err != nil {
		log.Fatalf("Error while read config: %s", err)
	}

	return &config
}
