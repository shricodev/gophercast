build:
	@go build -o bin/gophercast main.go

run: build
	@./bin/gophercast

serve: build
	@./bin/gophercast serve --dir-to-mp3 ~/Music

fmt:
	go fmt ./...

test:
	go test ./...

test-verbose:
	go test -v ./...

clean:
	rm -f bin/gophercast
