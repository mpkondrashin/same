package main

import (
	"flag"
	"log"
	"path/filepath"
	"strings"
)

type IgnoreList struct {
	masks []string
}

var _ flag.Value = &IgnoreList{}

func NewIgnoreList() *IgnoreList {
	return &IgnoreList{}
}

func (i *IgnoreList) String() string {
	return strings.Join(i.masks, ", ")
}

func (i *IgnoreList) Set(value string) error {
	i.masks = append(i.masks, value)
	return nil
}

func (i *IgnoreList) Ignore(name string) bool {
	for _, mask := range i.masks {
		matched, err := filepath.Match(mask, name)
		if err != nil {
			log.Fatalf("\"%s\": %v", mask, err)
		}
		if matched {
			return true
		}
	}
	return false
}
