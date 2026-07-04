.PHONY: build run test docker-build docker-up docker-down docker-run clean tidy

BINARY=bin/osto
MAIN=./cmd/cli

build:
	go build -o $(BINARY) $(MAIN)

run: build
	go run $(MAIN)

test:
	go test ./... -v -count=1

tidy:
	go mod tidy

clean:
	rm -rf bin/ dist/ *.db

docker-build:
	docker compose build

docker-up:
	docker compose up --build

docker-run:
	docker compose run --rm -it osto

docker-down:
	docker compose down

# Run tests inside a container (optional)
docker-test:
	docker compose run --rm --entrypoint go osto test ./... -v -count=1
