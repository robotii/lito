build:
	go build -o ./lito ./cmd/lito/

run: build
	./lito
