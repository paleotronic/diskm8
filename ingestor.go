package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"strings"

	"crypto/md5"

	"github.com/paleotronic/diskm8/disk"
	"github.com/paleotronic/diskm8/loggy"
	"github.com/paleotronic/diskm8/panic"
)

var diskRegex = regexp.MustCompile("(?i)[.](po|do|dsk)$")

func processFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		loggy.Get(0).Errorf(err.Error())
		return err
	}

	if diskRegex.MatchString(path) {

		incoming <- path

		fmt.Printf("\rIngested: %d volumes ...", processed)

	}

	return nil
}

const loaderWorkers = 8

var incoming chan string
var processed int
var errorcount int
var indisk map[disk.DiskFormat]int
var outdisk map[disk.DiskFormat]int
var cm sync.Mutex

func init() {
	indisk = make(map[disk.DiskFormat]int)
	outdisk = make(map[disk.DiskFormat]int)
}

func in(f disk.DiskFormat) {
	cm.Lock()
	indisk[f] = indisk[f] + 1
	cm.Unlock()
}

func out(f disk.DiskFormat) {
	cm.Lock()
	outdisk[f] = outdisk[f] + 1
	cm.Unlock()
}

func walk(dir string) {

	start := time.Now()

	incoming = make(chan string, 16)
	indisk = make(map[disk.DiskFormat]int)
	outdisk = make(map[disk.DiskFormat]int)

	var wg sync.WaitGroup
	var s sync.Mutex

	for i := 0; i < loaderWorkers; i++ {
		wg.Add(1)
		go func(i int) {

			id := 1 + i
			l := loggy.Get(id)

			for filename := range incoming {

				panic.Do(
					func() {
						analyze(id, filename)
						s.Lock()
						processed++
						s.Unlock()
					},
					func(r interface{}) {
						l.Errorf("Error processing volume: %s", filename)
						l.Errorf(string(debug.Stack()))
						s.Lock()
						errorcount++
						s.Unlock()
					},
				)

			}

			wg.Done()

		}(i)
	}

	filepath.Walk(dir, processFile)

	close(incoming)
	wg.Wait()

	fmt.Printf("\rIngested: %d volumes ...", processed)

	fmt.Println()

	duration := time.Since(start)

	fmt.Println("=============================================================")
	fmt.Printf(" DSKalyzer process report (%d Workers, %v)\n", loaderWorkers, duration)
	fmt.Println("=============================================================")

	tin, tout := 0, 0

	for f, count := range indisk {
		outcount := outdisk[f]
		fmt.Printf("%-30s %6d in %6d out\n", f.String(), count, outcount)
		tin += count
		tout += outcount
	}

	fmt.Println()

	fmt.Printf("%-30s %6d in %6d out\n", "Total", tin, tout)

	fmt.Println()

	average := duration / time.Duration(processed+errorcount)

	fmt.Printf("%v average time spent per disk.\n", average)
}

func existsPatternOld(base string, pattern string) (bool, []string) {

	l := loggy.Get(0)

	p := base + "/" + pattern

	l.Logf("glob: %s", p)

	matches, _ := filepath.Glob(p)

	return (len(matches) > 0), matches

}

func resolvePathfilters(base string, pathfilter []string, pattern string) []*regexp.Regexp {

	tmp := strings.Replace(pattern, ".", "[.]", -1)
	tmp = strings.Replace(tmp, "?", ".", -1)
	tmp = strings.Replace(tmp, "*", ".+", -1)
	tmp += "$"

	// pathfilter either contains filenames or a pattern (eg. if it was quoted)
	var out []*regexp.Regexp
	for _, p := range pathfilter {

		if runtime.GOOS == "windows" {
			//p = strings.Replace(p, ":", "", -1)
			p = strings.Replace(p, "\\", "/", -1)
		}

		//fmt.Printf("Stat [%s]\n", p)

		p, e := filepath.Abs(p)
		if e != nil {
			continue
		}

		//fmt.Printf("OK\n")

		// path is okay and now absolute
		info, e := os.Stat(p)
		if e != nil {
			continue
		}

		if runtime.GOOS == "windows" {
			p = strings.Replace(p, ":", "", -1)
			p = strings.Replace(p, "\\", "/", -1)
		}

		var realpath string
		if info.IsDir() {
			realpath = strings.Replace(base, "\\", "/", -1) + "/" + strings.Trim(p, "/") + "/" + tmp
		} else {
			// file
			b := strings.Trim(filepath.Base(p), " /")
			s := md5.Sum([]byte(b))
			realpath = strings.Replace(base, "\\", "/", -1) + "/" + strings.Trim(filepath.Dir(p), "/") + "/.+_.+_.+_" + hex.EncodeToString(s[:]) + "[.]fgp$"
		}

		//fmt.Printf("Regexp [%s]\n", realpath)

		out = append(out, regexp.MustCompile(realpath))

	}

	return out

}

func existsPattern(base string, filters []string, pattern string) (bool, []string) {

	tmp := strings.Replace(pattern, ".", "[.]", -1)
	tmp = strings.Replace(tmp, "?", ".", -1)
	tmp = strings.Replace(tmp, "*", ".+", -1)
	tmp = "(?i)" + tmp + "$"
	//os.Stderr.WriteString("Globby is: " + tmp + "\r\n")
	fileRxp := regexp.MustCompile(tmp)

	var out []string
	var found bool

	processPatternPath := func(path string, info os.FileInfo, err error) error {

		l := loggy.Get(0)

		if err != nil {
			l.Errorf(err.Error())
			return err
		}

		if fileRxp.MatchString(filepath.Base(path)) {
			found = true
			out = append(out, path)
		}

		return nil
	}

	filepath.Walk(base, processPatternPath)

	fexp := resolvePathfilters(base, filters, pattern)

	if len(fexp) > 0 {
		out2 := make([]string, 0)
		for _, p := range out {

			if runtime.GOOS == "windows" {
				p = strings.Replace(p, "\\", "/", -1)
			}

			for _, rxp := range fexp {
				//fmt.Printf("Match [%s]\n", p)
				if rxp.MatchString(p) {
					out2 = append(out2, p)
					//fmt.Printf("Match regexp: %s\n", p)
					break
				}
			}
		}
		//fmt.Printf("%d returns\n", len(out2))
		return (len(out2) > 0), out2
	}

	//fmt.Printf("%d returns\n", len(out))

	return found, out

}

func analyze(id int, filename string) (*Disk, error) {

	l := loggy.Get(id)

	var err error
	var dsk *disk.DSKWrapper
	var dskInfo Disk = Disk{}

	dskInfo.Filename = path.Base(filename)

	if abspath, e := filepath.Abs(filename); e == nil {
		filename = abspath
	}

	dskInfo.FullPath = path.Clean(filename)

	l.Logf("Reading disk image from file source %s", filename)
	//fmt.Printf("Processing %s\n", filename)
	//fmt.Print(".")

	dsk, err = disk.NewDSKWrapper(defNibbler, filename)

	if err != nil {
		l.Errorf("Disk read failed: %s", err)
		return &dskInfo, err
	}

	if dsk.Format.ID == disk.DF_DOS_SECTORS_13 || dsk.Format.ID == disk.DF_DOS_SECTORS_16 {
		isADOS, _, _ := dsk.IsAppleDOS()
		if !isADOS {
			dsk.Format.ID = disk.DF_NONE
			dsk.Layout = disk.SectorOrderDOS33
		}
	}
	// fmt.Printf("%s: IsAppleDOS=%v, Format=%s, Layout=%d\n", path.Base(filename), isADOS, Format, Layout)

	l.Log("Load is OK.")

	dskInfo.SHA256 = dsk.ChecksumDisk()
	l.Logf("SHA256 is %s", dskInfo.SHA256)

	dskInfo.Format = dsk.Format.String()
	dskInfo.FormatID = dsk.Format
	l.Logf("Format is %s", dskInfo.Format)

	l.Debugf("TOSO MAGIC: %v", hex.EncodeToString(dsk.Data[:32]))

	t, s := dsk.HuntVTOC(35, 13)
	l.Logf("Hunt VTOC says: %d, %d", t, s)

	// Check if it exists

	in(dsk.Format)

	dskInfo.IngestMode = *ingestMode

	switch dsk.Format.ID {
	case disk.DF_DOS_SECTORS_16:
		analyzeDOS16(id, dsk, &dskInfo)
	case disk.DF_DOS_SECTORS_13:
		analyzeDOS13(id, dsk, &dskInfo)
	case disk.DF_PRODOS_400KB:
		analyzePRODOS800(id, dsk, &dskInfo)
	case disk.DF_PRODOS_800KB:
		analyzePRODOS800(id, dsk, &dskInfo)
	case disk.DF_PRODOS:
		analyzePRODOS16(id, dsk, &dskInfo)
	case disk.DF_RDOS_3:
		analyzeRDOS(id, dsk, &dskInfo)
	case disk.DF_RDOS_32:
		analyzeRDOS(id, dsk, &dskInfo)
	case disk.DF_RDOS_33:
		analyzeRDOS(id, dsk, &dskInfo)
	case disk.DF_PASCAL:
		analyzePASCAL(id, dsk, &dskInfo)
	default:
		analyzeNONE(id, dsk, &dskInfo)
	}

	return &dskInfo, nil

}
