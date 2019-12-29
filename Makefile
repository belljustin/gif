build: clean
	go build -o build/server cmd/server/main.go

clean:
	rm -rf build/*
