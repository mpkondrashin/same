package main

import "io"

type SameFiles struct {
	files []*SameFile
}

func NewSameFiles() *SameFiles {
	return &SameFiles{}
}

func (s *SameFiles) AddFile(f *SameFile) {
	s.files = append(s.files, f)
}

func (s *SameFiles) Report(w io.Writer) {
	for _, f := range s.files {
		f.Report(w)
	}
}

func (s *SameFiles) WastedSpace() int64 {
	var wasted int64
	for _, f := range s.files {
		wasted += f.WastedSpace()
	}
	return wasted
}

func (s *SameFiles) Populate(fa *FixUp) {
	for _, sameFile := range s.files {
		sameFile.PopulateFixUp(fa)
	}
}
