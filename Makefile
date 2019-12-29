build: clean
	GOOS=linux GOARCH=386 go build -o build/server_linux_386 cmd/server/main.go
	GOOS=darwin GOARCH=386 go build -o build/server_darwin_386 cmd/server/main.go

clean:
	rm -rf build/*
