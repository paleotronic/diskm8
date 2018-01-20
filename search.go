package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type SearchResultContext int

const (
	SRC_UNKNOWN SearchResultContext = iota
	SRC_FILE
	SRC_DISK
)

type SearchResultItem struct {
	DiskPath string
	File     *DiskFile
}

func searchForFilename(filename string, filter []string) {

	fd := GetAllFiles("*_*_*_*.fgp", filter)

	fmt.Printf("Filter: %s\n", filter)

	fmt.Println()
	fmt.Println()

	fmt.Printf("SEARCH RESULTS FOR '%s'\n", filename)

	fmt.Println()

	for diskname, list := range fd {
		//fmt.Printf("Checking: %s\n", diskname)
		for _, f := range list {
			if strings.Contains(strings.ToLower(f.Filename), strings.ToLower(filename)) {
				fmt.Printf("%32s:\n  %s (%s, %d bytes, sha: %s)\n\n", diskname, f.Filename, f.Type, f.Size, f.SHA256)
				if *extract == "@" {
					ExtractFile(diskname, f, *adornedCP, false)
				} else if *extract == "#" {
					ExtractDisk(diskname)
				}
			}
		}
	}

}

func searchForSHA256(sha string, filter []string) {

	fd := GetAllFiles("*_*_*_*.fgp", filter)

	fmt.Println()
	fmt.Println()

	fmt.Printf("SEARCH RESULTS FOR SHA256 '%s'\n", sha)

	fmt.Println()

	for diskname, list := range fd {
		for _, f := range list {
			if f.SHA256 == sha {
				fmt.Printf("%32s:\n  %s (%s, %d bytes, sha: %s)\n\n", diskname, f.Filename, f.Type, f.Size, f.SHA256)
				if *extract == "@" {
					ExtractFile(diskname, f, *adornedCP, false)
				} else if *extract == "#" {
					ExtractDisk(diskname)
				}
			}
		}
	}

}

func searchForTEXT(text string, filter []string) {

	fd := GetAllFiles("*_*_*_*.fgp", filter)

	fmt.Println()
	fmt.Println()

	fmt.Printf("SEARCH RESULTS FOR TEXT CONTENT '%s'\n", text)

	fmt.Println()

	for diskname, list := range fd {
		for _, f := range list {
			if strings.Contains(strings.ToLower(string(f.Text)), strings.ToLower(text)) {
				fmt.Printf("%32s:\n  %s (%s, %d bytes, sha: %s)\n\n", diskname, f.Filename, f.Type, f.Size, f.SHA256)
				if *extract == "@" {
					ExtractFile(diskname, f, *adornedCP, false)
				} else if *extract == "#" {
					ExtractDisk(diskname)
				}
			}
		}
	}

}

func directory(filter []string, format string) {

	fd := GetAllFiles("*_*_*_*.fgp", filter)

	fmt.Println()
	fmt.Println()

	fmt.Println()

	for diskname, list := range fd {
		fmt.Printf("CATALOG RESULTS FOR '%s'\n", diskname)
		//fmt.Printf("Checking: %s\n", diskname)
		out := ""
		for _, file := range list {
			tmp := format
			// size
			tmp = strings.Replace(tmp, "{size:blocks}", fmt.Sprintf("%3d Blocks", file.Size/256+1), -1)
			tmp = strings.Replace(tmp, "{size:kb}", fmt.Sprintf("%4d Kb", file.Size/1024+1), -1)
			tmp = strings.Replace(tmp, "{size:b}", fmt.Sprintf("%6d Bytes", file.Size), -1)
			tmp = strings.Replace(tmp, "{size}", fmt.Sprintf("%6d", file.Size), -1)
			// format
			tmp = strings.Replace(tmp, "{filename}", fmt.Sprintf("%-36s", file.Filename), -1)
			// type
			tmp = strings.Replace(tmp, "{type}", fmt.Sprintf("%-20s", file.Type), -1)
			// sha256
			tmp = strings.Replace(tmp, "{sha256}", file.SHA256, -1)

			out += tmp + "\n"

			if *extract == "@" {
				ExtractFile(diskname, file, *adornedCP, false)
			} else if *extract == "#" {
				ExtractDisk(diskname)
			}
		}
		fmt.Println(out + "\n\n")
	}

}

var fileExtractCounter int

func ExtractFile(diskname string, fd *DiskFile, adorned bool, local bool) error {

	var name string

	if adorned {
		name = fd.GetNameAdorned()
	} else {
		name = fd.GetName()
	}

	path := binpath() + "/extract" + diskname

	if local {
		ext := filepath.Ext(diskname)
		base := strings.Replace(filepath.Base(diskname), ext, "", -1)
		path = "./" + base
	}

	if path != "." {
		os.MkdirAll(path, 0755)
	}

	//fmt.Printf("FD.EXT=%s\n", fd.Ext)

	f, err := os.Create(path + "/" + name)
	if err != nil {
		return err
	}
	defer f.Close()
	f.Write(fd.Data)
	os.Stderr.WriteString("Extracted file to " + path + "/" + name + "\n")

	if strings.ToLower(fd.Ext) == "int" || strings.ToLower(fd.Ext) == "bas" || strings.ToLower(fd.Ext) == "txt" {
		f, err := os.Create(path + "/" + name + ".ASC")
		if err != nil {
			return err
		}
		defer f.Close()
		f.Write(fd.Text)
		os.Stderr.WriteString("Extracted file to " + path + "/" + name + ".ASC\n")
	}

	//os.Stderr.WriteString("Extracted file to " + path + "/" + name)

	fileExtractCounter++

	return nil

}

func ExtractDisk(diskname string) error {
	path := binpath() + "/extract" + diskname
	os.MkdirAll(path, 0755)
	data, err := ioutil.ReadFile(diskname)
	if err != nil {
		return err
	}
	target := path + "/" + filepath.Base(diskname)
	return ioutil.WriteFile(target, data, 0755)
}
