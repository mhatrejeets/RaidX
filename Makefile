# Makefile for RaidX Go Project

run:
	go run .

build:
	go build -o raidx-server .

test:
	go test ./...

lint:
	golint ./...


seed:
	bash scripts/seed.sh

clean:
	rm -f raidx-server

.PHONY: run build test lint migrate clean
