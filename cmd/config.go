package main

import (
	"net/url"
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

var (
	logLevel    log.Level
	databaseURL *url.URL
	botToken    string
	authCode    string
)

type specification struct {
	LogLevel    string `envconfig:"LOG_LEVEL" default:"INFO"`
	DatabaseURL string `envconfig:"DATABASE_URL"`
	BotToken    string `envconfig:"TELEGRAM_BOT_TOKEN"`
	AuthCode    string `envconfig:"AUTH_CODE"`
}

func parseConfig() error {
	_ = godotenv.Load()

	config := new(specification)
	err := envconfig.Process("", config)
	if err != nil {
		return err
	}

	logLevel, err = log.ParseLevel(config.LogLevel)
	if err != nil {
		return err
	}
	log.SetFormatter(&log.TextFormatter{DisableSorting: false})
	log.SetOutput(os.Stdout)
	log.SetLevel(logLevel)

	if config.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is not provided")
	}
	databaseURL, err = url.Parse(config.DatabaseURL)
	if err != nil {
		return err
	}

	botToken = config.BotToken
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not provided")
	}

	authCode = config.AuthCode

	return nil
}
