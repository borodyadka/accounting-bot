//+build postgres

package main

import (
	"github.com/borodyadka/accounting-bot"
	"github.com/borodyadka/accounting-bot/storage/postgres"
)

func init() {
	registerRepository("postgres", func(url string) (accounting_bot.Repository, error) {
		return postgres.New(url)
	})
}
