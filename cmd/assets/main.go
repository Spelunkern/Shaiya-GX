// Command assets inspects the original data.sah/data.saf.
//
//	assets -data "$SHAIYA_DATA" stats
//	assets -data "$SHAIYA_DATA" ls interface
//	assets -data "$SHAIYA_DATA" cat interface/login.tga > login.tga
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/Spelunkern/shaiya-go-client/internal/data/archive"
)

func main() {
	dataDir := flag.String("data", os.Getenv("SHAIYA_DATA"), "directory with data.sah and data.saf")
	flag.Parse()
	if *dataDir == "" || flag.NArg() == 0 {
		log.Fatal("usage: assets -data DIR stats | ls [DIR] | cat FILE | find SUBSTR")
	}
	a, err := archive.Open(*dataDir)
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()
	switch flag.Arg(0) {
	case "stats":
		files, bytes := 0, int64(0)
		ext := map[string]int{}
		a.Walk(func(p string, f *archive.File) {
			files++
			bytes += int64(f.Length)
			if i := strings.LastIndexByte(p, '.'); i >= 0 {
				ext[strings.ToLower(p[i:])]++
			}
		})
		fmt.Printf("version %d, header count %d, %d files, %.1f MiB\n", a.Version, a.FileCount, files, float64(bytes)/(1<<20))
		keys := make([]string, 0, len(ext))
		for k := range ext {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return ext[keys[i]] > ext[keys[j]] })
		for _, k := range keys {
			fmt.Printf("  %-8s %d\n", k, ext[k])
		}
	case "ls":
		d, ok := a.LookupDir(flag.Arg(1))
		if !ok {
			log.Fatalf("no directory %q", flag.Arg(1))
		}
		for _, s := range d.Dirs {
			fmt.Printf("%s/\n", s.Name)
		}
		for _, f := range d.Files {
			fmt.Printf("%s\t%d\n", f.Name, f.Length)
		}
	case "cat":
		b, err := a.ReadFile(flag.Arg(1))
		if err != nil {
			log.Fatal(err)
		}
		os.Stdout.Write(b)
	case "find":
		needle := strings.ToLower(flag.Arg(1))
		a.Walk(func(p string, f *archive.File) {
			if strings.Contains(strings.ToLower(p), needle) {
				fmt.Printf("%s\t%d\n", p, f.Length)
			}
		})
	default:
		log.Fatalf("unknown command %q", flag.Arg(0))
	}
}
