package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
	"time"

	"github.com/inhies/go-bytesize"
)

const PreviewSize = 1024 * 2
const updateStatusInterval = 100 * time.Millisecond

var HashNew = md5.New

func main() {
	var reportFileName string
	flag.StringVar(&reportFileName, "report", "", "report file path")
	var logFileName string
	flag.StringVar(&logFileName, "log", "", "log file path")
	var scriptFileName string
	defaultScriptName := "rm.sh"
	if runtime.GOOS == "windows" {
		defaultScriptName = "rm.cmd"
	}
	flag.StringVar(&scriptFileName, "script", defaultScriptName, "remove duplicates script file path")
	var hashAlgorithm string
	flag.StringVar(&hashAlgorithm, "hash", "md5", "hash algorithm. Available values: md5, sha1, sha256")
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "verbose mode")
	ignoreList := NewIgnoreList()
	flag.Var(ignoreList, "ignore", "mask of files and folders to ignore")
	flag.Parse()
	log.Println("narg", flag.NArg())
	if flag.NArg() == 0 {
		log.Print("Missing folder command line parameter")
		fmt.Println("Usage: same [options] folder [folder...]")
		flag.Usage()
		os.Exit(1)
	}
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
		HashNew = md5.New
	case "sha1":
		HashNew = sha1.New
	case "sha256":
		HashNew = sha256.New
	default:
		log.Printf("Unsupported hash algorithm: %s", hashAlgorithm)
		flag.Usage()
		os.Exit(1)
	}
	fmt.Printf("process %s\n", flag.Args())
	log.Printf("process %s", flag.Args())
	app := NewSameApp(ignoreList)
	sf := NewSameFiles()
	err := app.Run(flag.Args(), sf)
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
	fixUp := NewFixUp()
	sf.Populate(fixUp)
	sh.WriteString(fixUp.ShellScript())
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
