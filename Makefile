# Makefile for RaidX Go Project

run:
	go run .

build:
	go build -o raidx-server .

test:
	go test ./...

lint:
	golint ./...

migrate:
	@echo "Run your DB migration command here"

clean:
	rm -f raidx-server

.PHONY: run build test lint migrate clean
