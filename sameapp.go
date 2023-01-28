package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type SameApp struct {
	lastStatus       time.Time
	fistPhaseCount   int
	secondPhaseCount int
	thirdPhaseCount  int
	counters         map[string]int
}

func NewSameApp() *SameApp {
	return &SameApp{
		counters: make(map[string]int),
	}
}

func (s *SameApp) Status() {
	now := time.Now()
	if s.lastStatus.Add(updateStatusInterval).After(now) {
		return
	}
	fmt.Printf("\r%d\t\t%d\t\t%d", s.fistPhaseCount, s.secondPhaseCount, s.thirdPhaseCount)
	s.lastStatus = now
}

func (s *SameApp) FirstPhase(roots []string) map[int64][]string {
	result := make(map[int64][]string)
	for _, root := range roots {
		_ = filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				log.Print(err)
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if info.Mode()&os.ModeSymlink == os.ModeSymlink {
				return nil
			}
			if info.Size() == 0 {
				return nil
			}
			if filepath.Base(path) == ".DS_Store" {
				return nil
			}
			s.fistPhaseCount++
			s.Status()
			result[info.Size()] = append(result[info.Size()], path)
			return nil
		})
	}
	//fmt.Printf("Phase 1: %d\n", s.fistPhaseCount)
	s.counters["1. Total files processed"] = s.fistPhaseCount
	// Remove unique files
	s.counters["2. Groups of the sames size"] = len(result)
	for size, paths := range result {
		if len(paths) == 1 {
			delete(result, size)
		}
	}
	s.counters["3. After removing groups with only one element"] = len(result)
	for _, paths := range result {
		s.counters["3.1 Resulting number of files"] += len(paths)
	}
	// Return the rest
	return result
}

type HashFunc func(path string) (string, error)

func (s *SameApp) FilterUsingHash(item int, paths []string, calcHash HashFunc) (map[string][]string, error) {
	result := make(map[string][]string)
	for _, path := range paths {
		hash, err := calcHash(path)
		if err != nil {
			log.Print(err)
			continue
		}
		result[hash] = append(result[hash], path)
	}
	s.counters[fmt.Sprintf("%d. Total number of files", 4+item*3)] += len(paths)
	s.counters[fmt.Sprintf("%d. Groups with same hash", 5+item*3)] += len(result)
	// Remove unique files
	for h, p := range result {
		if len(p) == 1 {
			delete(result, h)
		}
	}
	s.counters[fmt.Sprintf("%d. After deleting groups with only one element", 6+item*3)] += len(result)
	for _, paths := range result {
		s.counters[fmt.Sprintf("%d.1 Resulting number of files", 6+item*3)] += len(paths)
	}
	// Return the rest
	return result, nil
}

func (s *SameApp) SecondPhase(paths []string) (map[string][]string, error) {
	return s.FilterUsingHash(0, paths, func(path string) (string, error) {
		s.Status()
		s.secondPhaseCount++
		data, err := Preview(path)
		if err != nil {
			return "", err
		}
		return hex.EncodeToString(HashNew().Sum(data)), nil
	})
}

func (s *SameApp) ThirdPhase(paths []string) (map[string][]string, error) {
	return s.FilterUsingHash(1, paths, func(path string) (string, error) {
		s.Status()
		s.thirdPhaseCount++
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer f.Close()
		h := HashNew()
		_, err = io.Copy(h, f)
		if err != nil {
			return "", err
		}
		return hex.EncodeToString(h.Sum(nil)), nil
	})
}

func (s *SameApp) Run(root []string, sameFiles *SameFiles) error {
	fmt.Printf("Phase #1\tPhase #2\tPhase #3\n")
	data1 := s.FirstPhase(root)
	sizes := SortedSizes(data1)
	for _, size := range sizes {
		paths1 := data1[size]
		data2, err := s.SecondPhase(paths1)
		if err != nil {
			return err
		}
		for _, paths2 := range data2 {
			data3, err := s.ThirdPhase(paths2)
			if err != nil {
				return err
			}
			for hash, paths3 := range data3 {
				sf := NewSameFile(hash, size, nil)
				for _, path := range paths3 {
					sf.AddPath(path)
				}
				sameFiles.AddFile(sf)
			}
		}
	}
	return nil
}

func (s *SameApp) PrintCounters() {
	cNames := make([]string, 0)
	for name := range s.counters {
		cNames = append(cNames, name)
	}
	sort.Strings(cNames)
	for _, name := range cNames {
		value := s.counters[name]
		fmt.Printf("%s: %d\n", name, value)
	}
}
