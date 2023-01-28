package main

import (
	"fmt"
	"io"

	"github.com/inhies/go-bytesize"
)

type SameFile struct {
	hash  string
	size  int64
	paths []string
}

func NewSameFile(hash string, size int64, paths []string) *SameFile {
	return &SameFile{
		hash:  hash,
		size:  size,
		paths: paths,
	}
}

func (s *SameFile) AddPath(path string) {
	s.paths = append(s.paths, path)
}

func (f *SameFile) Report(w io.Writer) {
	fmt.Fprintf(w, "[%s] %s\n", f.hash, bytesize.New(float64(f.size)))
	for i, path := range f.paths {
		fmt.Fprintf(w, "[%d] %s\n", i+1, path)
	}
}

func (f *SameFile) WastedSpace() int64 {
	return f.size * int64(len(f.paths)-1)
}

func (f *SameFile) PopulateFixUp(fixUp *FixUp) {
	shortest := 0
	for i, path := range f.paths[1:] {
		if len(path) < len(f.paths[shortest]) {
			shortest = i + 1
		}
	}
	for i, path := range f.paths {
		if i == shortest {
			continue
		}
		fixUp.Add(path)
	}
}
