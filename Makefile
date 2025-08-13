install:
	go install -race go-racer/cmd/racer
build:
	go build -race -o bin go-racer/cmd/racer
