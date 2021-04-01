PROJECT = $(shell basename $(CURDIR))

.PHONY: build run traffic

build:
	go build -o build/$(PROJECT) cmd/$(PROJECT)/*

run: build
	build/ziplookup

traffic:
	echo "GET http://localhost:8080/90210" | vegeta attack -duration=5s | tee results.bin | vegeta report