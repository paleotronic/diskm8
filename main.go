package main

/*
DiskM8 is an open source offshoot of the file handling code from the Octalyzer
project.

It provides some command line tools for manipulating Apple // disk images, and
some work in progress reporting tools to ingest large quantities of files,
catalog them and detect duplicates.

The code currently needs a lot of refactoring and cleanup, which we will be working
through as time goes by.
*/

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"runtime/debug"

	"flag"

	"os"

	"github.com/paleotronic/diskm8/disk"
	"github.com/paleotronic/diskm8/loggy"

	"runtime"

	"strings"

	"time"

	"github.com/paleotronic/diskm8/panic"
)

func usage() {
	fmt.Printf(`%s <options>

Tool checks for duplicate or similar apple ][ disks, specifically those
with %d bytes size.	

`, path.Base(os.Args[0]), disk.STD_DISK_BYTES)
	flag.PrintDefaults()
}

func binpath() string {

	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE") + "/DiskM8"
	}
	return os.Getenv("HOME") + "/DiskM8"

}

func init() {
	loggy.LogFolder = binpath() + "/logs/"
}

var dskName = flag.String("ingest", "", "Disk file or path to ingest")
var dskInfo = flag.String("query", "", "Disk file to query or analyze")
var baseName = flag.String("datastore", binpath()+"/fingerprints", "Database of disk fingerprints for checking")
var verbose = flag.Bool("verbose", false, "Log to stderr")
var fileDupes = flag.Bool("file-dupes", false, "Run file dupe report")
var wholeDupes = flag.Bool("whole-dupes", false, "Run whole disk dupe report")
var activeDupes = flag.Bool("as-dupes", false, "Run active sectors only disk dupe report")
var asPartial = flag.Bool("as-partial", false, "Run partial active sector match against single disk (-disk required)")
var similarity = flag.Float64("similarity", 0.90, "Object match threshold for -*-partial reports")
var minSame = flag.Int("min-same", 0, "Minimum same # files for -all-file-partial")
var maxDiff = flag.Int("max-diff", 0, "Maximum different # files for -all-file-partial")
var filePartial = flag.Bool("file-partial", false, "Run partial file match against single disk (-disk required)")
var fileMatch = flag.String("file", "", "Search for other disks containing file")
var dir = flag.Bool("dir", false, "Directory specified disk (needs -disk)")
var dirFormat = flag.String("dir-format", "{filename} {type} {size:kb} Checksum: {sha256}", "Format of dir")
var preCache = flag.Bool("c", true, "Cache data to memory for quicker processing")
var allFilePartial = flag.Bool("all-file-partial", false, "Run partial file match against all disks")
var allSectorPartial = flag.Bool("all-sector-partial", false, "Run partial sector match (all) against all disks")
var activeSectorPartial = flag.Bool("active-sector-partial", false, "Run partial sector match (active only) against all disks")
var allFileSubset = flag.Bool("all-file-subset", false, "Run subset file match against all disks")
var activeSectorSubset = flag.Bool("active-sector-subset", false, "Run subset (active) sector match against all disks")
var allSectorSubset = flag.Bool("all-sector-subset", false, "Run subset (non-zero) sector match against all disks")
var filterPath = flag.Bool("select", false, "Select files for analysis or search based on file/dir/mask")
var csvOut = flag.Bool("csv", false, "Output data to CSV format")
var reportFile = flag.String("out", "", "Output file (empty for stdout)")
var catDupes = flag.Bool("cat-dupes", false, "Run duplicate catalog report")
var searchFilename = flag.String("search-filename", "", "Search database for file with name")
var searchSHA = flag.String("search-sha", "", "Search database for file with checksum")
var searchTEXT = flag.String("search-text", "", "Search database for file containing text")
var forceIngest = flag.Bool("force", false, "Force re-ingest disks that already exist")
var ingestMode = flag.Int("ingest-mode", 1, "Ingest mode:\n\t0=Fingerprints only\n\t1=Fingerprints + text\n\t2=Fingerprints + sector data\n\t3=All")
var extract = flag.String("extract", "", "Extract files/disks matched in searches ('#'=extract disk, '@'=extract files)")
var adornedCP = flag.Bool("adorned", true, "Extract files named similar to CP")
var shell = flag.Bool("shell", false, "Start interactive mode")
var shellBatch = flag.String("shell-batch", "", "Execute shell command(s) from file and exit")
var withDisk = flag.String("with-disk", "", "Perform disk operation (-file-extract,-file-put,-file-delete)")
var fileExtract = flag.String("file-extract", "", "File to delete from disk (-with-disk)")
var filePut = flag.String("file-put", "", "File to put on disk (-with-disk)")
var fileDelete = flag.String("file-delete", "", "File to delete (-with-disk)")
var fileMkdir = flag.String("dir-create", "", "Directory to create (-with-disk)")
var fileCatalog = flag.Bool("catalog", false, "List disk contents (-with-disk)")
var quarantine = flag.Bool("quarantine", false, "Run -as-dupes and -whole-disk in quarantine mode")

func main() {

	runtime.GOMAXPROCS(8)

	banner()

	//l.Default.Level = l.LevelCrit

	flag.Parse()

	var filterpath []string

	if *filterPath || *shell {
		for _, v := range flag.Args() {
			filterpath = append(filterpath, filepath.Clean(v))
		}
	}

	//l.SILENT = !*logToFile
	loggy.ECHO = *verbose

	if *withDisk != "" {
		dsk, err := disk.NewDSKWrapper(defNibbler, *withDisk)
		if err != nil {
			os.Stderr.WriteString(err.Error())
			os.Exit(2)
		}
		commandVolumes[0] = dsk
		commandTarget = 0
		switch {
		case *fileExtract != "":
			shellProcess("extract " + *fileExtract)
		case *filePut != "":
			shellProcess("put " + *filePut)
		case *fileMkdir != "":
			shellProcess("mkdir " + *fileMkdir)
		case *fileDelete != "":
			shellProcess("delete " + *fileDelete)
		case *fileCatalog:
			shellProcess("cat ")
		default:
			os.Stderr.WriteString("Additional flag required")
			os.Exit(3)
		}

		time.Sleep(5 * time.Second)

		os.Exit(0)
	}

	// if *preCache {
	// 	x := GetAllFiles("*_*_*_*.fgp")
	// 	fmt.Println(len(x))
	// }
	if *shellBatch != "" {
		var data []byte
		var err error
		if *shellBatch == "stdin" {
			data, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				os.Stderr.WriteString("Failed to read commands from stdin. Aborting")
				os.Exit(1)
			}
		} else {
			data, err = ioutil.ReadFile(*shellBatch)
			if err != nil {
				os.Stderr.WriteString("Failed to read commands from file. Aborting")
				os.Exit(1)
			}
		}
		lines := strings.Split(string(data), "\n")
		for i, l := range lines {
			r := shellProcess(l)
			if r == -1 {
				os.Stderr.WriteString(fmt.Sprintf("Script failed at line %d: %s\n", i+1, l))
				os.Exit(2)
			}
			if r == 999 {
				os.Stderr.WriteString("Script terminated")
				return
			}
		}
		return
	}

	if *shell {
		var dsk *disk.DSKWrapper
		var err error
		if len(filterpath) > 0 {
			fmt.Printf("Trying to load %s\n", filterpath[0])
			dsk, err = disk.NewDSKWrapper(defNibbler, filterpath[0])
			if err != nil {
				fmt.Println("Error: " + err.Error())
				os.Exit(1)
			}
		}
		shellDo(dsk)
		os.Exit(0)
	}

	defer func() {

		if fileExtractCounter > 0 {
			os.Stderr.WriteString(fmt.Sprintf("%d files were extracted\n", fileExtractCounter))
		}

	}()

	if *searchFilename != "" {
		searchForFilename(*searchFilename, filterpath)
		return
	}

	if *searchSHA != "" {
		searchForSHA256(*searchSHA, filterpath)
		return
	}

	if *searchTEXT != "" {
		searchForTEXT(*searchTEXT, filterpath)
		return
	}

	if *dir {
		directory(filterpath, *dirFormat)
		return
	}

	if *allFileSubset {
		allFilesSubsetReport(filterpath)
		os.Exit(0)
	}

	if *activeSectorSubset {
		activeSectorsSubsetReport(filterpath)
		os.Exit(0)
	}

	if *allSectorSubset {
		allSectorsSubsetReport(filterpath)
		os.Exit(0)
	}

	if *catDupes {
		allFilesPartialReport(1.0, filterpath, "DUPLICATE CATALOG REPORT")
		os.Exit(0)
	}

	if *allFilePartial {
		if *minSame == 0 && *maxDiff == 0 {
			allFilesPartialReport(*similarity, filterpath, "")
		} else if *minSame > 0 {
			allFilesCustomReport(keeperAtLeastNSame, filterpath, fmt.Sprintf("AT LEAST %d FILES MATCH", *minSame))
		} else if *maxDiff > 0 {
			allFilesCustomReport(keeperMaximumNDiff, filterpath, fmt.Sprintf("NO MORE THAN %d FILES DIFFER", *maxDiff))
		}
		os.Exit(0)
	}

	if *allSectorPartial {
		allSectorsPartialReport(*similarity, filterpath)
		os.Exit(0)
	}

	if *activeSectorPartial {
		activeSectorsPartialReport(*similarity, filterpath)
		os.Exit(0)
	}

	if *fileDupes {
		fileDupeReport(filterpath)
		os.Exit(0)
	}

	if *wholeDupes {
		if *quarantine {
			quarantineWholeDisks(filterpath)
		} else {
			wholeDupeReport(filterpath)
		}
		os.Exit(0)
	}

	if *activeDupes {
		if *quarantine {
			quarantineActiveDisks(filterpath)
		} else {
			activeDupeReport(filterpath)
		}
		os.Exit(0)
	}

	_, e := os.Stat(*baseName)
	if e != nil {
		loggy.Get(0).Logf("Creating path %s", *baseName)
		os.MkdirAll(*baseName, 0755)
	}

	if *dskName == "" && *dskInfo == "" {

		var dsk *disk.DSKWrapper
		var err error
		if len(filterpath) > 0 {
			fmt.Printf("Trying to load %s\n", filterpath[0])
			dsk, err = disk.NewDSKWrapper(defNibbler, filterpath[0])
			if err != nil {
				fmt.Println("Error: " + err.Error())
				os.Exit(1)
			}
		}
		shellDo(dsk)
		os.Exit(0)

	}

	info, err := os.Stat(*dskName)
	if err != nil {
		loggy.Get(0).Errorf("Error stating file: %s", err.Error())
		os.Exit(2)
	}
	if info.IsDir() {
		walk(*dskName)
	} else {
		indisk = make(map[disk.DiskFormat]int)
		outdisk = make(map[disk.DiskFormat]int)

		panic.Do(
			func() {
				dsk, e := analyze(0, *dskName)
				// handle any disk specific
				if e == nil && *asPartial {
					asPartialReport(dsk, *similarity, *reportFile, filterpath)
				} else if e == nil && *filePartial {
					filePartialReport(dsk, *similarity, *reportFile, filterpath)
				} else if e == nil && *fileMatch != "" {
					fileMatchReport(dsk, *fileMatch, filterpath)
				} else if e == nil && *dir {
					info := dsk.GetDirectory(*dirFormat)
					fmt.Printf("Directory of %s:\n\n", dsk.Filename)
					fmt.Println(info)
				}
			},
			func(r interface{}) {
				loggy.Get(0).Errorf("Error processing volume: %s", *dskName)
				loggy.Get(0).Errorf(string(debug.Stack()))
			},
		)
	}
}
