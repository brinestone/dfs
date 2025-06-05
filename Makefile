build:
	@go build -C cmd -o ..\bin\app.exe
build-nix:
	@go build -C cmd -o ../bin/app
run:
	@make build
	@".\build\app.exe"
test:
	@go test ./...
dev:
	@$$GOPATH/bin/air --build.cmd "make build" --build.bin ".\bin\app.exe" --build.exclude_dir "templates,build,bin,.git,node_modules"