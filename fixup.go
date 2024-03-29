package main

import (
	"fmt"
	"runtime"
	"strings"
)

type FixUp struct {
	files []string
}

func NewFixUp() *FixUp {
	return &FixUp{}
}

func (f *FixUp) Add(fileName string) {
	f.files = append(f.files, fileName)
}

func (f *FixUp) ShellScript() string {
	sb := new(strings.Builder)
	for _, f := range f.files {
		if runtime.GOOS == "windows" {
			fmt.Fprintf(sb, "del \"%s\"\r\n", f)
		} else {
			fmt.Fprintf(sb, "rm \"%s\"\n", f)
		}
	}
	return sb.String()
}
