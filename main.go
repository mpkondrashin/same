package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/inhies/go-bytesize"
)

const PreviewSize = 1024 * 2
const updateStatusInterval = 100 * time.Millisecond

var hashNew = md5.New

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

func (s *SameApp) Status(format string, v ...interface{}) {
	now := time.Now()
	if s.lastStatus.Add(updateStatusInterval).After(now) {
		return
	}
	//fmt.Printf(format, v...)
	fmt.Printf("\r%d\t\t%d\t\t%d", s.fistPhaseCount, s.secondPhaseCount, s.thirdPhaseCount)
	s.lastStatus = now
}

func (s *SameApp) FirstPhase(root string) map[int64][]string {
	result := make(map[int64][]string)
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
		s.Status("Phase 1: %d\n", s.fistPhaseCount)
		result[info.Size()] = append(result[info.Size()], path)
		return nil
	})
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
		s.Status("Phase 2: %d\n", s.secondPhaseCount)
		s.secondPhaseCount++
		data, err := Preview(path)
		if err != nil {
			return "", err
		}
		return hex.EncodeToString(hashNew().Sum(data)), nil
	})
}

func (s *SameApp) ThirdPhase(paths []string) (map[string][]string, error) {
	return s.FilterUsingHash(1, paths, func(path string) (string, error) {
		s.Status("Phase 3: %d\n", s.thirdPhaseCount)
		s.thirdPhaseCount++
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer f.Close()
		h := hashNew()
		_, err = io.Copy(h, f)
		if err != nil {
			return "", err
		}
		return hex.EncodeToString(h.Sum(nil)), nil
	})
}

func (s *SameApp) Run(root string, sameFiles *SameFiles) error {
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

func main() {
	fs := flag.NewFlagSet("arguments", flag.ExitOnError)
	var reportFileName string
	fs.StringVar(&reportFileName, "report", "", "report file path")
	var logFileName string
	fs.StringVar(&logFileName, "log", "", "log file path")
	var scriptFileName string
	fs.StringVar(&scriptFileName, "script", "rm.sh", "remove duplicates script file path")
	var hashAlgorithm string
	fs.StringVar(&hashAlgorithm, "hash", "md5", "hash algorithm. Available values: md5, sha1, sha256")
	var verbose bool
	fs.BoolVar(&verbose, "verbose", false, "verbose mode")
	if len(os.Args) < 2 {
		log.Print("Missing folder command line parameter")
		fmt.Println("Usage: same folder")
		fs.Usage()
		os.Exit(1)
	}
	fs.Parse(os.Args[2:])
	if logFileName != "" {
		f, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Log file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
		log.Println("Same started")
	}
	switch hashAlgorithm {
	case "md5":
		hashNew = md5.New
	case "sha1":
		hashNew = sha1.New
	case "sha256":
		hashNew = sha256.New
	default:
		log.Printf("Unsupported hash algorithm: %s", hashAlgorithm)
		fs.Usage()
		os.Exit(1)
	}
	root := os.Args[1]
	fmt.Printf("process %s\n", root)
	log.Printf("process %s", root)
	app := NewSameApp()
	sf := NewSameFiles()
	err := app.Run(root, sf)
	if err != nil {
		panic(err)
	}
	if reportFileName != "" {
		repFile, err := os.Create(reportFileName)
		if err != nil {
			log.Print(err)
		} else {
			sf.Report(repFile)
			repFile.Close()
		}
	} else {
		sf.Report(os.Stdout)
	}
	fmt.Println()
	if verbose {
		app.PrintCounters()
	}
	fmt.Printf("Wasted space: %v\n", bytesize.New(float64(sf.WastedSpace())))
	sh, err := os.Create(scriptFileName)
	if err != nil {
		panic(err)
	}
	defer sh.Close()
	sh.WriteString(sf.FixUp().ShellScript())
	if err != nil {
		fmt.Print(err)
		return
	}
	fmt.Printf("Remove doubles script: %s\n", scriptFileName)
}

func Preview(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, PreviewSize)
	_, err = f.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func SortedSizes(data map[int64][]string) []int64 {
	sizes := make([]int64, len(data))
	i := 0
	for size := range data {
		sizes[i] = size
		i++
	}
	sort.SliceStable(sizes, func(i, j int) bool {
		return sizes[i] < sizes[j]
	})
	return sizes
}

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
	//sb := new(strings.Builder)
	fmt.Fprintf(w, "[%s] %s\n", f.hash, bytesize.New(float64(f.size)))
	for i, path := range f.paths {
		fmt.Fprintf(w, "[%d] %s\n", i+1, path)
	}
	//return sb.String()
}

func (f *SameFile) WastedSpace() int64 {
	return f.size * int64(len(f.paths)-1)
}

func (f *SameFile) FixUp(fixUp *FixUp) {
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

func (s *SameFiles) FixUp() *FixUp {
	fa := NewFixUp()
	for _, sameFile := range s.files {
		sameFile.FixUp(fa)
	}
	return fa
}

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
		fmt.Fprintf(sb, "rm \"%s\"\n", f)
	}
	return sb.String()
}
