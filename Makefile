EXE := aws-key-rotator
VER := $(shell git describe --tags)

.PHONY: release clean test darwin linux windows

$(EXE): go.mod go.sum *.go
	go build -v -ldflags '-X main.Version=$(VER)' -o $@

release: $(EXE) darwin windows linux

darwin linux:
	GOOS=$@ go build -ldflags '-X main.Version=$(VER)' -o $(EXE)-$(VER)-$@-amd64
windows:
	GOOS=$@ go build -ldflags '-X main.Version=$(VER)' -o $(EXE)-$(VER)-$@-amd64.exe

clean:
	rm -f $(EXE) $(EXE)-*-*

test:
	go test -v ./...
