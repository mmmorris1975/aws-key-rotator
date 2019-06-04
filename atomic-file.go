package main

import "os"

// AtomicFile represents a file created with exclusive mode for use as a simple lock file
type AtomicFile struct {
	*os.File
}

// NewAtomicFile returns an AtomicFile at the specified path, which supports all methods of an os.File type
func NewAtomicFile(path string) (*AtomicFile, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	return &AtomicFile{f}, nil
}
