.PHONY: run watch

ifneq (,$(wildcard .env))
	include .env
	export
endif

all: build

build: bin/bot

bin/bot: $(./.../*.go)
	go build -mod vendor -tags postgres -o bin/bot ./cmd/

run:
	go run -mod vendor ./cmd

watch:
	reflex -s -r '.go$$' -- go run -mod vendor -tags postgres ./cmd
