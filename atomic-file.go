package main

import "os"

type AtomicFile struct {
	*os.File
}

func NewAtomicFile(path string) (*AtomicFile, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	return &AtomicFile{f}, nil
}
