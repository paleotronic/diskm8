package main

import (
	"fmt"
	"os"
	"sort"
)

type DuplicateSource struct {
	Fullpath    string
	Filename    string
	GSHA        string
	fingerprint string
}

type DuplicateFileCollection struct {
	data map[string][]DuplicateSource
}

type DuplicateWholeDiskCollection struct {
	data map[string][]DuplicateSource
}

type DuplicateActiveSectorDiskCollection struct {
	data    map[string][]DuplicateSource
	data_as map[string][]DuplicateSource
}

func (dfc *DuplicateFileCollection) Add(checksum string, fullpath string, filename string, fgp string) {

	if dfc.data == nil {
		dfc.data = make(map[string][]DuplicateSource)
	}

	list, ok := dfc.data[checksum]
	if !ok {
		list = make([]DuplicateSource, 0)
	}

	list = append(list, DuplicateSource{Fullpath: fullpath, Filename: filename, fingerprint: fgp})

	dfc.data[checksum] = list

}

func (dfc *DuplicateWholeDiskCollection) Add(checksum string, fullpath string, fgp string) {

	if dfc.data == nil {
		dfc.data = make(map[string][]DuplicateSource)
	}

	list, ok := dfc.data[checksum]
	if !ok {
		list = make([]DuplicateSource, 0)
	}

	list = append(list, DuplicateSource{Fullpath: fullpath, fingerprint: fgp})

	dfc.data[checksum] = list

}

func (dfc *DuplicateActiveSectorDiskCollection) Add(checksum string, achecksum string, fullpath string, fgp string) {

	if dfc.data == nil {
		dfc.data = make(map[string][]DuplicateSource)
	}

	list, ok := dfc.data[achecksum]
	if !ok {
		list = make([]DuplicateSource, 0)
	}

	list = append(list, DuplicateSource{Fullpath: fullpath, GSHA: checksum, fingerprint: fgp})

	dfc.data[achecksum] = list

}

func (dfc *DuplicateFileCollection) Report(filename string) {

	var w *os.File
	var err error

	if filename != "" {
		w, err = os.Create(filename)
		if err != nil {
			return
		}
		defer w.Close()
	} else {
		w = os.Stdout
	}

	for sha256, list := range dfc.data {

		if len(list) > 1 {

			w.WriteString(fmt.Sprintf("\nChecksum %s duplicated %d times:\n", sha256, len(list)))
			for i, v := range list {
				w.WriteString(fmt.Sprintf(" %d) %s >> %s\n", i, v.Fullpath, v.Filename))
			}

		}

	}

}

func AggregateDuplicateFiles(d *Disk, collection interface{}) {

	for _, f := range d.Files {

		collection.(*DuplicateFileCollection).Add(f.SHA256, d.FullPath, f.Filename, d.source)

	}

}

func AggregateDuplicateWholeDisks(d *Disk, collection interface{}) {

	collection.(*DuplicateWholeDiskCollection).Add(d.SHA256, d.FullPath, d.source)

}

func AggregateDuplicateActiveSectorDisks(d *Disk, collection interface{}) {

	collection.(*DuplicateActiveSectorDiskCollection).Add(d.SHA256, d.SHA256Active, d.FullPath, d.source)

}

func (dfc *DuplicateWholeDiskCollection) Report(filename string) {

	var disksWithDupes int
	var extras int

	var w *os.File
	var err error

	if filename != "" {
		w, err = os.Create(filename)
		if err != nil {
			return
		}
		defer w.Close()
	} else {
		w = os.Stdout
	}

	for sha256, list := range dfc.data {

		if len(list) > 1 {

			disksWithDupes++

			original := list[0]
			dupes := list[1:]

			w.WriteString("\n")
			w.WriteString(fmt.Sprintf("Volume %s has %d duplicate(s):\n", original.Fullpath, len(dupes)))
			for _, v := range dupes {
				w.WriteString(fmt.Sprintf(" %s (sha256: %s)\n", v.Fullpath, sha256))
				extras++
			}

		}

	}

	w.WriteString("\n")
	w.WriteString("SUMMARY\n")
	w.WriteString("=======\n")
	w.WriteString(fmt.Sprintf("Total disks which have duplicates: %d\n", disksWithDupes))
	w.WriteString(fmt.Sprintf("Total redundant copies found     : %d\n", extras))

}

func (dfc *DuplicateActiveSectorDiskCollection) Report(filename string) {

	var disksWithDupes int
	var extras int

	var w *os.File
	var err error

	if filename != "" {
		w, err = os.Create(filename)
		if err != nil {
			return
		}
		defer w.Close()
	} else {
		w = os.Stdout
	}

	for sha256, list := range dfc.data {

		if len(list) > 1 {

			m := make(map[string]int)
			for _, v := range list {
				m[v.GSHA] = 1
			}

			if len(m) == 1 {
				continue
			}

			disksWithDupes++

			original := list[0]
			dupes := list[1:]

			w.WriteString("\n")
			w.WriteString("--------------------------------------\n")
			w.WriteString(fmt.Sprintf("Volume       : %s\n", original.Fullpath))
			w.WriteString(fmt.Sprintf("Active SHA256: %s\n", sha256))
			w.WriteString(fmt.Sprintf("Global SHA256: %s\n", original.GSHA))
			w.WriteString(fmt.Sprintf("# Duplicates : %d\n", len(dupes)))
			for i, v := range dupes {
				w.WriteString("\n")
				w.WriteString(fmt.Sprintf(" Duplicate #%d\n", i+1))
				w.WriteString(fmt.Sprintf(" = Volume       : %s\n", v.Fullpath))
				w.WriteString(fmt.Sprintf(" = Active SHA256: %s\n", sha256))
				w.WriteString(fmt.Sprintf(" = Global SHA256: %s\n", v.GSHA))
				extras++
			}
			w.WriteString("\n")

		}

	}

	w.WriteString("\n")
	w.WriteString("SUMMARY\n")
	w.WriteString("=======\n")
	w.WriteString(fmt.Sprintf("Total disks which have duplicates: %d\n", disksWithDupes))
	w.WriteString(fmt.Sprintf("Total redundant copies found     : %d\n", extras))

}

func asPartialReport(d *Disk, t float64, filename string, pathfilter []string) {
	matches := d.GetPartialMatchesWithThreshold(t, pathfilter)

	var w *os.File
	var err error

	if filename != "" {
		w, err = os.Create(filename)
		if err != nil {
			return
		}
		defer w.Close()
	} else {
		w = os.Stdout
	}

	w.WriteString(fmt.Sprintf("PARTIAL ACTIVE SECTOR MATCH REPORT FOR %s (Above %.2f%%)\n\n", d.Filename, 100*t))

	//sort.Sort(ByMatchFactor(matches))
	sort.Sort(ByMatchFactor(matches))

	w.WriteString(fmt.Sprintf("%d matches found\n\n", len(matches)))
	for i := len(matches) - 1; i >= 0; i-- {
		v := matches[i]

		w.WriteString(fmt.Sprintf("%.2f%%\t%s\n", v.MatchFactor*100, v.FullPath))

	}

	w.WriteString("")
}

func filePartialReport(d *Disk, t float64, filename string, pathfilter []string) {
	matches := d.GetPartialFileMatchesWithThreshold(t, pathfilter)

	var w *os.File
	var err error

	if filename != "" {
		w, err = os.Create(filename)
		if err != nil {
			return
		}
		defer w.Close()
	} else {
		w = os.Stdout
	}

	w.WriteString(fmt.Sprintf("PARTIAL FILE MATCH REPORT FOR %s (Above %.2f%%)\n\n", d.Filename, 100*t))

	//sort.Sort(ByMatchFactor(matches))
	sort.Sort(ByMatchFactor(matches))

	w.WriteString(fmt.Sprintf("%d matches found\n\n", len(matches)))
	for i := len(matches) - 1; i >= 0; i-- {
		v := matches[i]

		w.WriteString(fmt.Sprintf("%.2f%%\t%s (%d missing, %d extras)\n", v.MatchFactor*100, v.FullPath, len(v.MissingFiles), len(v.ExtraFiles)))
		for f1, f2 := range v.MatchFiles {
			w.WriteString(fmt.Sprintf("\t == %s -> %s\n", f1.Filename, f2.Filename))
		}
		for _, f := range v.MissingFiles {
			w.WriteString(fmt.Sprintf("\t -- %s\n", f.Filename))
		}
		for _, f := range v.ExtraFiles {
			w.WriteString(fmt.Sprintf("\t ++ %s\n", f.Filename))
		}
		w.WriteString("")

	}

	w.WriteString("")
}

func fileMatchReport(d *Disk, filename string, pathfilter []string) {

	matches := d.GetFileMatches(filename, pathfilter)

	var w *os.File
	var err error

	if filename != "" {
		w, err = os.Create(filename)
		if err != nil {
			return
		}
		defer w.Close()
	} else {
		w = os.Stdout
	}

	w.WriteString(fmt.Sprintf("PARTIAL FILE MATCH REPORT FOR %s (File: %s)\n\n", d.Filename, filename))

	w.WriteString(fmt.Sprintf("%d matches found\n\n", len(matches)))
	for i, v := range matches {

		w.WriteString(fmt.Sprintf("%d)\t%s\n", i, v.FullPath))
		for f1, f2 := range v.MatchFiles {
			w.WriteString(fmt.Sprintf("\t == %s -> %s\n", f1.Filename, f2.Filename))
		}
		w.WriteString("")

	}

	w.WriteString("")
}

func fileDupeReport(filter []string) {

	dfc := &DuplicateFileCollection{}
	Aggregate(AggregateDuplicateFiles, dfc, filter)

	fmt.Println("DUPLICATE FILE REPORT")
	fmt.Println()

	dfc.Report(*reportFile)

}

func wholeDupeReport(filter []string) {

	dfc := &DuplicateWholeDiskCollection{}
	Aggregate(AggregateDuplicateWholeDisks, dfc, filter)

	fmt.Println("DUPLICATE WHOLE DISK REPORT")
	fmt.Println()

	dfc.Report(*reportFile)

}

func activeDupeReport(filter []string) {

	dfc := &DuplicateActiveSectorDiskCollection{}
	Aggregate(AggregateDuplicateActiveSectorDisks, dfc, filter)

	fmt.Println("DUPLICATE ACTIVE SECTORS DISK REPORT")
	fmt.Println()

	dfc.Report(*reportFile)

}

func allFilesPartialReport(t float64, filter []string, oheading string) {

	matches := CollectFilesOverlapsAboveThreshold(t, filter)

	if *csvOut {
		dumpFileOverlapCSV(matches, *reportFile)
		return
	}

	if oheading != "" {
		fmt.Println(oheading + "\n")
	} else {
		fmt.Printf("PARTIAL ALL FILE MATCH REPORT (Above %.2f%%)\n\n", 100*t)
	}

	fmt.Printf("%d matches found\n\n", len(matches))
	for volumename, matchdata := range matches {

		fmt.Printf("Disk: %s\n", volumename)

		for k, ratio := range matchdata.percent {
			fmt.Println()
			fmt.Printf("  :: %.2f%% Match to %s\n", 100*ratio, k)
			for f1, f2 := range matchdata.files[k] {
				fmt.Printf("     == %s -> %s\n", f1.Filename, f2.Filename)
			}
			for _, f := range matchdata.missing[k] {
				fmt.Printf("     -- %s\n", f.Filename)
			}
			for _, f := range matchdata.extras[k] {
				fmt.Printf("     ++ %s\n", f.Filename)
			}
			fmt.Println()
		}

		fmt.Println()

	}

	fmt.Println()
}

func allSectorsPartialReport(t float64, filter []string) {

	matches := CollectSectorOverlapsAboveThreshold(t, filter, GetAllDiskSectors)

	if *csvOut {
		dumpSectorOverlapCSV(matches, *reportFile)
		return
	}

	fmt.Printf("NON-ZERO SECTOR MATCH REPORT (Above %.2f%%)\n\n", 100*t)

	fmt.Printf("%d matches found\n\n", len(matches))
	for volumename, matchdata := range matches {

		fmt.Printf("Disk: %s\n", volumename)

		for k, ratio := range matchdata.percent {
			fmt.Println()
			fmt.Printf("  :: %.2f%% Match to %s\n", 100*ratio, k)
			fmt.Printf("     == %d Sectors matched\n", len(matchdata.same[k]))
			fmt.Printf("     -- %d Sectors missing\n", len(matchdata.missing[k]))
			fmt.Printf("     ++ %d Sectors extra\n", len(matchdata.extras[k]))
			fmt.Println()
		}

		fmt.Println()

	}

	fmt.Println()
}

func activeSectorsPartialReport(t float64, filter []string) {

	matches := CollectSectorOverlapsAboveThreshold(t, filter, GetActiveDiskSectors)

	if *csvOut {
		dumpSectorOverlapCSV(matches, *reportFile)
		return
	}

	fmt.Printf("PARTIAL ACTIVE SECTOR MATCH REPORT (Above %.2f%%)\n\n", 100*t)

	fmt.Printf("%d matches found\n\n", len(matches))
	for volumename, matchdata := range matches {

		fmt.Printf("Disk: %s\n", volumename)

		for k, ratio := range matchdata.percent {
			fmt.Println()
			fmt.Printf("  :: %.2f%% Match to %s\n", 100*ratio, k)
			fmt.Printf("     == %d Sectors matched\n", len(matchdata.same[k]))
			fmt.Printf("     -- %d Sectors missing\n", len(matchdata.missing[k]))
			fmt.Printf("     ++ %d Sectors extra\n", len(matchdata.extras[k]))
			fmt.Println()
		}

		fmt.Println()

	}

	fmt.Println()
}

func allFilesSubsetReport(filter []string) {

	matches := CollectFileSubsets(filter)

	if *csvOut {
		dumpFileOverlapCSV(matches, *reportFile)
		return
	}

	fmt.Printf("SUBSET DISK FILE MATCH REPORT\n\n")

	fmt.Printf("%d matches found\n\n", len(matches))
	for volumename, matchdata := range matches {

		fmt.Printf("Disk: %s\n", volumename)

		for k, _ := range matchdata.percent {
			fmt.Println()
			fmt.Printf("  :: Is a file subset of %s\n", k)
			for f1, f2 := range matchdata.files[k] {
				fmt.Printf("     == %s -> %s\n", f1.Filename, f2.Filename)
			}
			for _, f := range matchdata.missing[k] {
				fmt.Printf("     -- %s\n", f.Filename)
			}
			for _, f := range matchdata.extras[k] {
				fmt.Printf("     ++ %s\n", f.Filename)
			}
			fmt.Println()
		}

		fmt.Println()

	}

	fmt.Println()
}

func activeSectorsSubsetReport(filter []string) {

	matches := CollectSectorSubsets(filter, GetActiveDiskSectors)

	if *csvOut {
		dumpSectorOverlapCSV(matches, *reportFile)
		return
	}

	fmt.Printf("ACTIVE SECTOR SUBSET MATCH REPORT\n\n")

	fmt.Printf("%d matches found\n\n", len(matches))
	for volumename, matchdata := range matches {

		fmt.Printf("Disk: %s\n", volumename)

		for k, _ := range matchdata.percent {
			fmt.Println()
			fmt.Printf("  :: Is a subset (based on active sectors) of %s\n", k)
			fmt.Printf("     == %d Sectors matched\n", len(matchdata.same[k]))
			fmt.Printf("     ++ %d Sectors extra\n", len(matchdata.extras[k]))
			fmt.Println()
		}

		fmt.Println()

	}

	fmt.Println()
}

func allSectorsSubsetReport(filter []string) {

	matches := CollectSectorSubsets(filter, GetAllDiskSectors)

	if *csvOut {
		dumpSectorOverlapCSV(matches, *reportFile)
		return
	}

	fmt.Printf("NON-ZERO SECTOR SUBSET MATCH REPORT\n\n")

	fmt.Printf("%d matches found\n\n", len(matches))
	for volumename, matchdata := range matches {

		fmt.Printf("Disk: %s\n", volumename)

		for k, _ := range matchdata.percent {
			fmt.Println()
			fmt.Printf("  :: Is a subset (based on active sectors) of %s\n", k)
			fmt.Printf("     == %d Sectors matched\n", len(matchdata.same[k]))
			fmt.Printf("     ++ %d Sectors extra\n", len(matchdata.extras[k]))
			fmt.Println()
		}

		fmt.Println()

	}

	fmt.Println()
}

func dumpFileOverlapCSV(matches map[string]*FileOverlapRecord, filename string) {

	var w *os.File
	var err error

	if filename != "" {
		w, err = os.Create(filename)
		if err != nil {
			return
		}
		defer w.Close()
	} else {
		w = os.Stderr
	}

	w.WriteString("MATCH,DISK1,FILENAME1,DISK2,FILENAME2,EXISTS\n")
	for disk1, matchdata := range matches {
		for disk2, match := range matchdata.percent {
			for f1, f2 := range matchdata.files[disk2] {
				w.WriteString(fmt.Sprintf(`%.2f,"%s","%s","%s","%s",%s`, match, disk1, f1.Filename, disk2, f2.Filename, "Y") + "\n")
			}
			for _, f1 := range matchdata.missing[disk2] {
				w.WriteString(fmt.Sprintf(`%.2f,"%s","%s","%s","%s",%s`, match, disk1, f1.Filename, disk2, "", "N") + "\n")
			}
			for _, f2 := range matchdata.extras[disk2] {
				w.WriteString(fmt.Sprintf(`%.2f,"%s","%s","%s","%s",%s`, match, disk1, "", disk2, f2.Filename, "N") + "\n")
			}
		}
	}

	if filename != "" {
		fmt.Println("\nWrote " + filename + "\n")
	}

}

func dumpSectorOverlapCSV(matches map[string]*SectorOverlapRecord, filename string) {

	var w *os.File
	var err error

	if filename != "" {
		w, err = os.Create(filename)
		if err != nil {
			return
		}
		defer w.Close()
	} else {
		w = os.Stderr
	}

	w.WriteString("MATCH,DISK1,DISK2,SAME,MISSING,EXTRA\n")
	for disk1, matchdata := range matches {
		for disk2, match := range matchdata.percent {
			w.WriteString(fmt.Sprintf(`%.2f,"%s","%s",%d,%d,%d`, match, disk1, disk2, len(matchdata.same[disk2]), len(matchdata.missing[disk2]), len(matchdata.extras[disk2])) + "\n")
		}
	}

	if filename != "" {
		fmt.Println("\nWrote " + filename + "\n")
	}

}

func keeperAtLeastNSame(d1, d2 string, v *FileOverlapRecord) bool {

	return len(v.files[d2]) >= *minSame

}

func keeperMaximumNDiff(d1, d2 string, v *FileOverlapRecord) bool {

	return len(v.files[d2]) > 0 && (len(v.missing[d2])+len(v.extras[d2])) <= *maxDiff

}

func allFilesCustomReport(keep func(d1, d2 string, v *FileOverlapRecord) bool, filter []string, oheading string) {

	matches := CollectFilesOverlapsCustom(keep, filter)

	if *csvOut {
		dumpFileOverlapCSV(matches, *reportFile)
		return
	}

	fmt.Println(oheading + "\n")

	fmt.Printf("%d matches found\n\n", len(matches))
	for volumename, matchdata := range matches {

		fmt.Printf("Disk: %s\n", volumename)

		for k, ratio := range matchdata.percent {
			fmt.Println()
			fmt.Printf("  :: %.2f%% Match to %s\n", 100*ratio, k)
			for f1, f2 := range matchdata.files[k] {
				fmt.Printf("     == %s -> %s\n", f1.Filename, f2.Filename)
			}
			for _, f := range matchdata.missing[k] {
				fmt.Printf("     -- %s\n", f.Filename)
			}
			for _, f := range matchdata.extras[k] {
				fmt.Printf("     ++ %s\n", f.Filename)
			}
			fmt.Println()
		}

		fmt.Println()

	}

	fmt.Println()
}
