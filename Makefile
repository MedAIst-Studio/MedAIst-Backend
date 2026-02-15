SWAG=$(HOME)/go/bin/swag

swagger: 
	@echo "Generating swagger docs..."
	$(SWAG) fmt
	$(SWAG) init -g infrastructure/fiber/router.go -d . --parseDependency --parseInternal -o ./docs
dev: swagger
	docker compose up 

build:
	go build -o  ./main ./cmd/main.go
	
run: 
	go run ./cmd/main.go

test: unit-test integration-test httptest e2e-test

unit-test:
	go test -v ./tests/unit/... 

integration-test:
	go test -v  ./tests/integration/...

httptest:
	go test -v ./tests/httptest/...

e2e-test:
	go test -v ./tests/e2e/...

