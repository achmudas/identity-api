.PHONY: run build test lint fmt tidy generate migrate

run:        ## run Service A locally
	go run ./cmd/api

build:      ## compile the static binary (used by the Dockerfile later)
	CGO_ENABLED=0 go build -o bin/api ./cmd/api

test:       ## unit + integration tests with the race detector
	go test -race -cover ./...

lint:       ## what CI runs too
	golangci-lint run

fmt:        ## auto-format (v2 runs formatters via `fmt`, not `run`)
	golangci-lint fmt

tidy:       ## sync go.mod to actual imports
	go mod tidy

generate:   ## runs sqlc (Day 5) and buf (Day 10) via //go:generate directives
	go generate ./...

migrate:    ## apply DB migrations (Day 5)
	migrate -path ./db/migrations -database "$(DB_URL)" up

run-profile:
	go run ./cmd/profile/