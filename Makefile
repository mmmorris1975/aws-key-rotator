EXE := aws-key-rotator
VER := $(shell git describe --tags)

.PHONY: release clean test darwin linux windows

$(EXE): Gopkg.lock *.go
	go build -v -i $@

Gopkg.lock: Gopkg.toml
	dep ensure

release: $(EXE) darwin windows linux

darwin linux:
	GOOS=$@ go build -o $(EXE)-$(VER)-$@
windows:
	GOOS=$@ go build -o $(EXE)-$(VER)-$@.exe

clean:
	rm -f $(EXE) $(EXE)-*-*

test:
	go test -v ./...
