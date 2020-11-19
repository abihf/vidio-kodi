default: build

build: build-amd64

build-all: build-amd64 build-arm7

build-amd64: dist/vidio-amd64

dist/vidio-amd64:
	mkdir -p dist
	CGO=0 GOOS=linux GOARCH=amd64 go build -o dist/vidio-amd64

build-arm7: dist/vidio-arm7

dist/vidio-arm7:
	mkdir -p dist
	CGO=0 GOOS=linux GOARCH=arm GOARM=7 go build -o dist/vidio-arm7
