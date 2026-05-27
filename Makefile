.PHONY: generate run build clean

generate:
	@echo "Generating code from OpenAPI spec..."
	@go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen \
		-package gen \
		-generate types,server,spec \
		-o api/gen/server.gen.go \
		api/openapi.yaml

run: generate
	@echo "Starting server..."
	@go run cmd/server/main.go

build: generate
	@echo "Building server..."
	@go build -o bin/server cmd/server/main.go

clean:
	@rm -rf bin/
	@rm -f api/gen/*.go
