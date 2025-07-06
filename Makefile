build:
	go build -o bin/gophercast main.go

run:
	go run main.go serve --dir ~/Music

fmt:
	go fmt ./...

test:
	go test ./...

# install-deps:
		# In future
