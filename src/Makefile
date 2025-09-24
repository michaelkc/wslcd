
# Build statically linked Linux binaries (no cgo)
# You can tweak GOARCH=amd64/arm64 as needed.

APP := wslcd

all: build

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 		go build -trimpath -tags "netgo osusergo" 		-ldflags "-s -w -extldflags=-static" 		-o $(APP) ./cmd/wslcd

build-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 		go build -trimpath -tags "netgo osusergo" 		-ldflags "-s -w -extldflags=-static" 		-o $(APP)-arm64 ./cmd/wslcd

install: build
	install -m 0755 $(APP) /usr/local/bin/$(APP)

clean:
	rm -f $(APP) $(APP)-arm64
