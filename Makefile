.PHONY: generate run build clean deb-repo

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
	@rm -rf debian-repo/
	@rm -rf deb-packages/

deb-repo:
	@echo "Building Debian repository..."
	@mkdir -p deb-packages
	@echo "Place .deb packages in deb-packages/ directory"
	@chmod +x scripts/build-debian-repo.sh
	@./scripts/build-debian-repo.sh deb-packages debian-repo
