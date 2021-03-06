package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	accbot "github.com/borodyadka/accounting-bot"
	"github.com/sirupsen/logrus"
)

type repositoryFactory func(url string) (accbot.Repository, error)

var repositories = make(map[string]repositoryFactory)

func registerRepository(provider string, factory repositoryFactory) {
	repositories[provider] = factory
}

func main() {
	if err := parseConfig(); err != nil {
		accbot.NewLogger(logrus.ErrorLevel, "bot").Fatal(err)
	}
	logger := accbot.NewLogger(logLevel, "bot")

	factory, sok := repositories[databaseURL.Scheme]
	if !sok {
		logger.Fatalf(`unknown database type "%s"`, databaseURL.Scheme)
	}
	storage, err := factory(databaseURL.String())
	if err != nil {
		logger.Fatal(err)
	}

	bot, err := accbot.New(botToken, logger, storage, botConfig)
	if err != nil {
		logger.Fatal(err)
	}

	errC := make(chan error, 1)
	go func(bot *accbot.Bot) {
		if err := bot.Start(); err != nil {
			errC <- err
		}
	}(bot)

	sigC := make(chan error, 1)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
		sigC <- fmt.Errorf("received signal %s", <-c)
	}()

	select {
	case sig := <-sigC:
		bot.Stop()
		logger.Info(sig)
		return
	case err := <-errC:
		logger.Fatal("error ", err)
		return
	}
}
