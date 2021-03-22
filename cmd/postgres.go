//+build postgres

package main

import (
	"github.com/borodyadka/accounting-bot"
	"github.com/borodyadka/accounting-bot/storage/postgres"
)

func init() {
	registerStorage("postgres", func(url string) (accounting_bot.Storage, error) {
		return postgres.New(url)
	})
}
