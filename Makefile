build:
	@if exist "..\bin\app.exe" del "..\bin\app.exe"
	@go build -C cmd -o ..\bin\app.exe
build-nix:
	@rm -f ../bin/app
	@go build -C cmd -o ../bin/app
run:
	@make build
	@".\build\app.exe -v"
test:
	@go test ./...
dev:
	@$$GOPATH/bin/air --build.cmd "make build" --build.bin ".\bin\app.exe" --build.args_bin "-v" --build.exclude_dir "templates,build,bin,.git,node_modules"
dev-nix:
	@$$GOPATH/bin/air --build.cmd "make build" --build.bin ".\bin\app.exe" --build.exclude_dir "templates,build,bin,.git,node_modules"