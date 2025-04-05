all: run

build:
	@mkdir -p bin
	@go build -o ./bin/connex ./cmd/connex/

run: build
	@./bin/connex

clean: 
	@rm -rf ./bin
