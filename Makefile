build:
	@rm -rf bin
	@go build -o bin/app

build-wasm:
	GOOS=wasip1 GOARCH=wasm go build -o bin/app.wasm

run:
	@make build
	@./bin/app

test:
	@go test ./...

dev:
	@$$GOPATH/bin/air --build.cmd "make build" --build.bin "./bin/app --n :4001,:4002 --p 4000" --build.exclude_dir "templates,build,bin"