build:
	go build -o bin/gophercast main.go

run:
	go run main.go serve --dir ~/Music

fmt:
	go fmt ./...

# install-deps:
		# In future
