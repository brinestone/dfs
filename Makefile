build:
	@rm -rf bin
	@go build -C cmd -o ../bin/app

build-wasm:
	GOOS=wasip1 GOARCH=wasm go build -o bin/app.wasm

run:
	@make build
	@./bin/app

test:
	@go test ./... -v

dev:
	@$$GOPATH/bin/air --build.cmd "make build" --build.bin "./bin/app" --build.exclude_dir "templates,build,bin"