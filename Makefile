EXE := aws-key-rotator
VER := $(shell git describe --tags)

.PHONY: release clean test darwin linux windows

$(EXE): *.go
	go build -v -i $@

release: $(EXE) darwin windows linux

darwin linux:
	GOOS=$@ go build -o $(EXE)-$(VER)-$@
windows:
	GOOS=$@ go build -o $(EXE)-$(VER)-$@.exe

clean:
	rm -f $(EXE) $(EXE)-*-*

test:
	go test -v ./...
