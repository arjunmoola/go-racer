.PHONY = install build clean

install:
	go install github.com/arjunmoola/go-racer/cmd/racer
build:
	go build -o bin github.com/arjunmoola/go-racer/cmd/racer
clean:
	rm bin/*
