package main

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	"crypto/md5"
	"encoding/hex"

	"os"

	"strings"

	"encoding/gob"

	"path/filepath"

	"github.com/paleotronic/diskm8/disk"
	"github.com/paleotronic/diskm8/loggy"
)

type Disk struct {
	FullPath                string
	Filename                string
	SHA256                  string // Sha of whole disk
	SHA256Active            string // Sha of active sectors/blocks only
	Format                  string
	FormatID                disk.DiskFormat
	Bitmap                  []bool
	Tracks, Sectors, Blocks int
	Files                   DiskCatalog
	ActiveSectors           DiskSectors
	//ActiveBlocks             DiskBlocks
	InactiveSectors DiskSectors
	//InactiveBlocks           DiskBlocks
	MatchFactor              float64
	MatchFiles               map[*DiskFile]*DiskFile
	MissingFiles, ExtraFiles []*DiskFile
	IngestMode               int
	source                   string
}

type ByMatchFactor []*Disk

func (s ByMatchFactor) Len() int {
	return len(s)
}

func (s ByMatchFactor) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ByMatchFactor) Less(i, j int) bool {
	return s[i].MatchFactor < s[j].MatchFactor
}

type TypeCode int

const (
	TypeMask_AppleDOS TypeCode = 0x0000
	TypeMask_ProDOS   TypeCode = 0x0100
	TypeMask_Pascal   TypeCode = 0x0200
	TypeMask_RDOS     TypeCode = 0x0300
)

type DiskFile struct {
	Filename    string
	Type        string
	Ext         string
	TypeCode    TypeCode
	SHA256      string
	Size        int
	LoadAddress int
	Text        []byte
	Data        []byte
	Locked      bool
	Created     time.Time
	Modified    time.Time
}

func (d *DiskFile) GetNameAdorned() string {

	var ext string
	switch d.TypeCode & 0xff00 {
	case TypeMask_AppleDOS:
		ext = disk.FileType(d.TypeCode & 0xff).Ext()
	case TypeMask_ProDOS:
		ext = disk.ProDOSFileType(d.TypeCode & 0xff).Ext()
	case TypeMask_RDOS:
		ext = disk.RDOSFileType(d.TypeCode & 0xff).Ext()
	case TypeMask_Pascal:
		ext = disk.PascalFileType(d.TypeCode & 0xff).Ext()
	}

	return fmt.Sprintf("%s#0x%.4x.%s", d.Filename, d.LoadAddress, ext)

}

func (d *DiskFile) GetName() string {

	var ext string
	switch d.TypeCode & 0xff00 {
	case TypeMask_AppleDOS:
		ext = disk.FileType(d.TypeCode & 0xff).Ext()
	case TypeMask_ProDOS:
		ext = disk.ProDOSFileType(d.TypeCode & 0xff).Ext()
	case TypeMask_RDOS:
		ext = disk.RDOSFileType(d.TypeCode & 0xff).Ext()
	case TypeMask_Pascal:
		ext = disk.PascalFileType(d.TypeCode & 0xff).Ext()
	}

	return fmt.Sprintf("%s.%s", d.Filename, ext)

}

type DiskCatalog []*DiskFile
type DiskSectors []*DiskSector
type DiskBlocks []*DiskBlock

type DiskSector struct {
	Track  int
	Sector int

	SHA256 string

	Data []byte
}

type DiskBlock struct {
	Block int

	SHA256 string
}

func (i Disk) LogBitmap(id int) {

	l := loggy.Get(id)

	if i.Tracks > 0 {

		for t := 0; t < i.Tracks; t++ {

			line := fmt.Sprintf("Track %.2d: ", t)

			for s := 0; s < i.Sectors; s++ {
				if i.Bitmap[t*i.Sectors+s] {
					line += fmt.Sprintf("%.2x ", s)
				} else {
					line += ":: "
				}
			}

			l.Logf("%s", line)
		}

	} else if i.Blocks > 0 {

		tr := i.Blocks / 16
		sc := 16

		for t := 0; t < tr; t++ {

			line := fmt.Sprintf("Block %.2d: ", t)

			for s := 0; s < sc; s++ {
				if i.Bitmap[t*i.Sectors+s] {
					line += fmt.Sprintf("%.2x ", s)
				} else {
					line += ":: "
				}
			}

			l.Logf("%s", line)
		}

	}

}

func (d Disk) GetFilename() string {

	sum := md5.Sum([]byte(d.Filename))

	//	fmt.Printf("checksum: [%s] -> [%s]\n", d.Filename, hex.EncodeToString(sum[:]))

	ff := fmt.Sprintf("%s/%d", strings.Trim(filepath.Dir(d.FullPath), "/"), d.FormatID.ID) + "_" + d.SHA256 + "_" + d.SHA256Active + "_" + hex.EncodeToString(sum[:]) + ".fgp"

	if runtime.GOOS == "windows" {
		ff = strings.Replace(ff, ":", "", -1)
		ff = strings.Replace(ff, "\\", "/", -1)
	}

	return ff

}

func (d Disk) WriteToFile(filename string) error {

	// b, err := yaml.Marshal(d)

	// if err != nil {
	// 	return err
	// }
	l := loggy.Get(0)

	_ = os.MkdirAll(filepath.Dir(filename), 0755)

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	enc.Encode(d)

	l.Logf("Created %s", filename)

	return nil
}

func (d *Disk) ReadFromFile(filename string) error {
	// b, err := ioutil.ReadFile(filename)
	// if err != nil {
	// 	return err
	// }
	// err = yaml.Unmarshal(b, d)

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	err = dec.Decode(d)

	d.source = filename

	return err
}

// GetExactBinaryMatches returns disks with the same Global SHA256
func (d *Disk) GetExactBinaryMatches(filter []string) []*Disk {

	l := loggy.Get(0)

	var out []*Disk = make([]*Disk, 0)

	exists, matches := existsPattern(*baseName, filter, fmt.Sprintf("%d", d.FormatID)+"_"+d.SHA256+"_*_*.fgp")
	if !exists {
		return out
	}

	for _, m := range matches {
		l.Logf(":: Checking %s", m)
		if item, err := cache.Get(m); err == nil {
			if item.FullPath != d.FullPath {
				out = append(out, item)
			}
		}
	}

	return out
}

// GetActiveSectorBinaryMatches returns disks with the same Active SHA256
func (d *Disk) GetActiveSectorBinaryMatches(filter []string) []*Disk {

	l := loggy.Get(0)

	var out []*Disk = make([]*Disk, 0)

	exists, matches := existsPattern(*baseName, filter, fmt.Sprintf("%d", d.FormatID)+"_*_"+d.SHA256Active+"_*.fgp")
	if !exists {
		return out
	}

	for _, m := range matches {
		l.Logf(":: Checking %s", m)

		if item, err := cache.Get(m); err == nil {
			if item.FullPath != d.FullPath {
				out = append(out, item)
			}
		}
	}

	return out
}

func (d *Disk) GetFileMap() map[string]*DiskFile {

	out := make(map[string]*DiskFile)

	for _, file := range d.Files {
		f := file
		out[file.SHA256] = f
	}

	return out

}

func (d *Disk) GetUtilizationMap() map[string]string {

	out := make(map[string]string)

	if len(d.ActiveSectors) > 0 {

		for _, block := range d.ActiveSectors {

			key := fmt.Sprintf("T%dS%d", block.Track, block.Sector)
			out[key] = block.SHA256

		}

	}

	return out

}

// CompareChunks returns a value 0-1
func (d *Disk) CompareChunks(b *Disk) (float64, float64, float64, float64) {

	l := loggy.Get(0)

	if d.FormatID != b.FormatID {
		l.Logf("Trying to compare disks of different types")
		return 0, 0, 0, 0
	}

	switch d.FormatID.ID {
	case disk.DF_RDOS_3:
		return d.compareSectorsPositional(b)
	case disk.DF_RDOS_32:
		return d.compareSectorsPositional(b)
	case disk.DF_RDOS_33:
		return d.compareSectorsPositional(b)
	case disk.DF_PASCAL:
		return d.compareBlocksPositional(b)
	case disk.DF_DOS_SECTORS_13:
		return d.compareSectorsPositional(b)
	case disk.DF_DOS_SECTORS_16:
		return d.compareSectorsPositional(b)
	case disk.DF_PRODOS:
		return d.compareBlocksPositional(b)
	case disk.DF_PRODOS_800KB:
		return d.compareBlocksPositional(b)
	}

	return 0, 0, 0, 0

}

func (d *Disk) compareSectorsPositional(b *Disk) (float64, float64, float64, float64) {

	l := loggy.Get(0)

	var sameSectors float64
	var diffSectors float64
	var dNotb float64
	var bNotd float64
	var emptySectors float64
	var dTotal, bTotal float64

	var dmap = d.GetUtilizationMap()
	var bmap = b.GetUtilizationMap()

	for t := 0; t < d.FormatID.TPD(); t++ {

		for s := 0; s < d.FormatID.SPT(); s++ {

			key := fmt.Sprintf("T%dS%d", t, s)

			dCk, dEx := dmap[key]
			bCk, bEx := bmap[key]

			switch {
			case dEx && bEx:
				if dCk == bCk {
					sameSectors += 1
				} else {
					diffSectors += 1
				}
				dTotal += 1
				bTotal += 1
			case dEx && !bEx:
				dNotb += 1
				dTotal += 1
			case !dEx && bEx:
				bNotd += 1
				bTotal += 1
			default:
				emptySectors += 1
			}

		}

	}

	l.Debugf("Same Sectors     : %f", sameSectors)
	l.Debugf("Differing Sectors: %f", diffSectors)
	l.Debugf("Not in other disk: %f", dNotb)
	l.Debugf("Not in this disk : %f", bNotd)

	// return sameSectors / dTotal, sameSectors / bTotal, diffSectors / dTotal, diffSectors / btotal
	return sameSectors / dTotal, sameSectors / bTotal, diffSectors / dTotal, diffSectors / bTotal

}

func (d *Disk) compareBlocksPositional(b *Disk) (float64, float64, float64, float64) {

	l := loggy.Get(0)

	var sameSectors float64
	var diffSectors float64
	var dNotb float64
	var bNotd float64
	var emptySectors float64
	var dTotal, bTotal float64

	var dmap = d.GetUtilizationMap()
	var bmap = b.GetUtilizationMap()

	for t := 0; t < d.FormatID.BPD(); t++ {

		key := fmt.Sprintf("B%d", t)

		dCk, dEx := dmap[key]
		bCk, bEx := bmap[key]

		switch {
		case dEx && bEx:
			if dCk == bCk {
				sameSectors += 1
			} else {
				diffSectors += 1
			}
			dTotal += 1
			bTotal += 1
		case dEx && !bEx:
			dNotb += 1
			dTotal += 1
		case !dEx && bEx:
			bNotd += 1
			bTotal += 1
		default:
			emptySectors += 1
		}

	}

	l.Debugf("Same Blocks      : %f", sameSectors)
	l.Debugf("Differing Blocks : %f", diffSectors)
	l.Debugf("Not in other disk: %f", dNotb)
	l.Debugf("Not in this disk : %f", bNotd)

	// return sameSectors / dTotal, sameSectors / bTotal, diffSectors / dTotal, diffSectors / btotal
	return sameSectors / dTotal, sameSectors / bTotal, diffSectors / dTotal, diffSectors / bTotal

}

// GetActiveSectorBinaryMatches returns disks with the same Active SHA256
func (d *Disk) GetPartialMatches(filter []string) ([]*Disk, []*Disk, []*Disk) {

	l := loggy.Get(0)

	var superset []*Disk = make([]*Disk, 0)
	var subset []*Disk = make([]*Disk, 0)
	var identical []*Disk = make([]*Disk, 0)

	exists, matches := existsPattern(*baseName, filter, fmt.Sprintf("%d", d.FormatID)+"_*_*_*.fgp")
	if !exists {
		return superset, subset, identical
	}

	for _, m := range matches {
		//item := &Disk{}
		if item, err := cache.Get(m); err == nil {
			if item.FullPath != d.FullPath {
				// only here if not looking at same disk
				l.Logf(":: Checking overlapping data blocks %s", item.Filename)
				l.Log("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
				dSame, iSame, dDiff, iDiff := d.CompareChunks(item)
				l.Logf("== This disk shares %.2f percent of its allocated blocks with %s", dSame*100, item.Filename)
				l.Logf("!= This disk differs %.2f percent of its allocate blocks with %s", dDiff*100, item.Filename)
				l.Logf("== %s shares %.2f of its blocks with this disk", item.Filename, iSame*100)
				l.Logf("!= %s differs %.2f of its blocks with this disk", item.Filename, iDiff*100)

				if dSame == 1 && iSame < 1 {
					superset = append(superset, item)
				} else if iSame == 1 && dSame < 1 {
					subset = append(subset, item)
				} else if iSame == 1 && dSame == 1 {
					identical = append(identical, item)
				}
			}
		}
	}

	return superset, subset, identical
}

func (d *Disk) GetPartialMatchesWithThreshold(t float64, filter []string) []*Disk {

	l := loggy.Get(0)

	var matchlist []*Disk = make([]*Disk, 0)

	exists, matches := existsPattern(*baseName, filter, fmt.Sprintf("%d", d.FormatID)+"_*_*_*.fgp")
	if !exists {
		return matchlist
	}

	var lastPc int = -1
	for i, m := range matches {
		//item := &Disk{}
		if item, err := cache.Get(m); err == nil {

			pc := int(100 * float64(i) / float64(len(matches)))

			if pc != lastPc {
				os.Stderr.WriteString(fmt.Sprintf("Analyzing volumes... %d%%   ", pc))
			}

			if item.FullPath != d.FullPath {
				// only here if not looking at same disk
				l.Logf(":: Checking overlapping data blocks %s", item.Filename)
				// l.Log("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
				dSame, _, _, _ := d.CompareChunks(item)
				// l.Logf("== This disk shares %.2f percent of its allocated blocks with %s", dSame*100, item.Filename)
				// l.Logf("!= This disk differs %.2f percent of its allocate blocks with %s", dDiff*100, item.Filename)
				// l.Logf("== %s shares %.2f of its blocks with this disk", item.Filename, iSame*100)
				// l.Logf("!= %s differs %.2f of its blocks with this disk", item.Filename, iDiff*100)

				item.MatchFactor = dSame

				if dSame >= t {
					matchlist = append(matchlist, item)
				}
			}

			fmt.Print("\r")
			lastPc = pc
		}
	}

	return matchlist
}

func Aggregate(f func(d *Disk, collector interface{}), collector interface{}, pathfilter []string) {

	l := loggy.Get(0)

	exists, matches := existsPattern(*baseName, pathfilter, "*_*_*_*.fgp")
	if !exists {
		return
	}

	var lastPc int = -1
	for i, m := range matches {

		pc := int(100 * float64(i) / float64(len(matches)))

		if pc != lastPc {
			os.Stderr.WriteString(fmt.Sprintf("\rAggregating data... %d%%   ", pc))
		}

		l.Logf(":: Checking %s", m)
		//item := &Disk{}
		if item, err := cache.Get(m); err == nil {
			f(item, collector)
		}

	}

	os.Stderr.WriteString("Done.\n")

	return
}

func (d *Disk) CompareFiles(b *Disk) float64 {

	var sameFiles float64
	var missingFiles float64
	var extraFiles float64

	var dmap = d.GetFileMap()
	var bmap = b.GetFileMap()

	for fileCk, info := range dmap {

		if info.Size == 0 {
			continue
		}

		binfo, bEx := bmap[fileCk]

		if bEx {
			sameFiles += 1
			// file match
			if b.MatchFiles == nil {
				b.MatchFiles = make(map[*DiskFile]*DiskFile)
			}
			//fmt.Printf("*** %s: %s -> %s\n", b.Filename, binfo.Filename, info.Filename)
			b.MatchFiles[binfo] = info
		} else {
			missingFiles += 1
			// file match
			if b.MissingFiles == nil {
				b.MissingFiles = make([]*DiskFile, 0)
			}
			//fmt.Printf("*** %s: %s -> %s\n", b.Filename, binfo.Filename, info.Filename)
			b.MissingFiles = append(b.MissingFiles, info)
		}

	}

	for fileCk, info := range bmap {

		if info.Size == 0 {
			continue
		}

		_, dEx := dmap[fileCk]

		if !dEx {
			extraFiles += 1
			// file match
			if b.ExtraFiles == nil {
				b.ExtraFiles = make([]*DiskFile, 0)
			}
			//fmt.Printf("*** %s: %s -> %s\n", b.Filename, binfo.Filename, info.Filename)
			b.ExtraFiles = append(b.ExtraFiles, info)
		}

	}

	// return sameSectors / dTotal, sameSectors / bTotal, diffSectors / dTotal, diffSectors / btotal
	return sameFiles / (sameFiles + extraFiles + missingFiles)

}

func (d *Disk) GetPartialFileMatchesWithThreshold(t float64, filter []string) []*Disk {

	l := loggy.Get(0)

	var matchlist []*Disk = make([]*Disk, 0)

	exists, matches := existsPattern(*baseName, filter, "*_*_*_*.fgp")
	if !exists {
		return matchlist
	}

	var lastPc int = -1
	for i, m := range matches {
		//item := &Disk{}
		if item, err := cache.Get(m); err == nil {

			pc := int(100 * float64(i) / float64(len(matches)))

			if pc != lastPc {
				os.Stderr.WriteString(fmt.Sprintf("Analyzing volumes... %d%%   ", pc))
			}

			if item.FullPath != d.FullPath {
				// only here if not looking at same disk
				l.Logf(":: Checking overlapping files %s", item.Filename)
				dSame := d.CompareFiles(item)

				item.MatchFactor = dSame

				if dSame >= t {
					matchlist = append(matchlist, item)
				}
			}

			fmt.Print("\r")
			lastPc = pc
		}
	}

	return matchlist
}

func (d *Disk) HasFileSHA256(sha string) (bool, *DiskFile) {

	for _, file := range d.Files {
		if sha == file.SHA256 {
			return true, file
		}
	}

	return false, nil

}

func (d *Disk) GetFileChecksum(filename string) (bool, string) {

	for _, f := range d.Files {
		if strings.ToLower(filename) == strings.ToLower(f.Filename) {
			return true, f.SHA256
		}
	}

	return false, ""

}

func (d *Disk) GetFileMatches(filename string, filter []string) []*Disk {

	l := loggy.Get(0)

	var matchlist []*Disk = make([]*Disk, 0)

	exists, matches := existsPattern(*baseName, filter, "*_*_*_*.fgp")
	if !exists {
		return matchlist
	}

	fileexists, SHA256 := d.GetFileChecksum(filename)
	if !fileexists {
		os.Stderr.WriteString("File does not exist on this volume: " + filename + "\n")
		return matchlist
	}

	_, srcFile := d.HasFileSHA256(SHA256)

	var lastPc int = -1
	for i, m := range matches {
		//item := &Disk{}
		if item, err := cache.Get(m); err == nil {

			pc := int(100 * float64(i) / float64(len(matches)))

			if pc != lastPc {
				os.Stderr.WriteString(fmt.Sprintf("Analyzing volumes... %d%%   ", pc))
			}

			if item.FullPath != d.FullPath {
				// only here if not looking at same disk
				l.Logf(":: Checking overlapping files %s", item.Filename)

				if ex, file := item.HasFileSHA256(SHA256); ex {
					if item.MatchFiles == nil {
						item.MatchFiles = make(map[*DiskFile]*DiskFile)
					}
					item.MatchFiles[srcFile] = file
					matchlist = append(matchlist, item)
				}
			}

			fmt.Print("\r")
			lastPc = pc
		}
	}

	return matchlist
}

// Gets directory with custom format
func (d *Disk) GetDirectory(format string) string {
	out := ""

	for _, file := range d.Files {

		tmp := format
		// size
		tmp = strings.Replace(tmp, "{size:blocks}", fmt.Sprintf("%3d Blocks", file.Size/256+1), -1)
		tmp = strings.Replace(tmp, "{size:kb}", fmt.Sprintf("%4d Kb", file.Size/1024+1), -1)
		tmp = strings.Replace(tmp, "{size:b}", fmt.Sprintf("%6d Bytes", file.Size), -1)
		tmp = strings.Replace(tmp, "{size}", fmt.Sprintf("%6d", file.Size), -1)
		// format
		tmp = strings.Replace(tmp, "{filename}", fmt.Sprintf("%-20s", file.Filename), -1)
		// type
		tmp = strings.Replace(tmp, "{type}", fmt.Sprintf("%-20s", file.Type), -1)
		// sha256
		tmp = strings.Replace(tmp, "{sha256}", file.SHA256, -1)
		// loadaddress
		tmp = strings.Replace(tmp, "{loadaddr}", fmt.Sprintf("0x.%4X", file.LoadAddress), -1)

		out += tmp + "\n"
	}

	return out
}

type CacheContext int

const (
	CC_All CacheContext = iota
	CC_ActiveSectors
	CC_AllSectors
	CC_Files
)

type DiskMetaDataCache struct {
	ctx   CacheContext
	Disks map[string]*Disk
}

var cache = NewCache(CC_All, "")

func (c *DiskMetaDataCache) Get(filename string) (*Disk, error) {
	cached, ok := c.Disks[filename]
	if ok {
		return cached, nil
	}
	item := &Disk{}
	if err := item.ReadFromFile(filename); err == nil {
		c.Disks[filename] = item
		return item, nil
	}
	return nil, errors.New("Not found")
}

func (c *DiskMetaDataCache) Put(filename string, item *Disk) {
	c.Disks[filename] = item
}

func NewCache(ctx CacheContext, pattern string) *DiskMetaDataCache {

	cache := &DiskMetaDataCache{
		ctx:   ctx,
		Disks: make(map[string]*Disk),
	}

	return cache
}

func CreateCache(ctx CacheContext, pattern string, filter []string) *DiskMetaDataCache {

	cache := &DiskMetaDataCache{
		ctx:   ctx,
		Disks: make(map[string]*Disk),
	}

	exists, matches := existsPattern(*baseName, filter, pattern)
	if !exists {
		return cache
	}

	var lastPc int = -1
	for i, m := range matches {
		item := &Disk{}
		if err := item.ReadFromFile(m); err == nil {

			pc := int(100 * float64(i) / float64(len(matches)))

			if pc != lastPc {
				os.Stderr.WriteString(fmt.Sprintf("Caching data... %d%%   ", pc))
			}

			// Load cache
			cache.Put(m, item)

			fmt.Print("\r")
			lastPc = pc
		}
	}

	return cache
}

func SearchPartialFileMatchesWithThreshold(t float64, filter []string) map[string][2]*Disk {

	l := loggy.Get(0)

	matchlist := make(map[string][2]*Disk)

	exists, matches := existsPattern(*baseName, filter, "*_*_*_*.fgp")
	if !exists {
		return matchlist
	}

	done := make(map[string]bool)

	var lastPc int = -1
	for i, m := range matches {
		//item := &Disk{}
		if disk, err := cache.Get(m); err == nil {

			d := *disk

			pc := int(100 * float64(i) / float64(len(matches)))

			if pc != lastPc {
				os.Stderr.WriteString(fmt.Sprintf("Analyzing volumes... %d%%   ", pc))
			}

			for _, n := range matches {

				if jj, err := cache.Get(n); err == nil {

					item := *jj

					key := d.SHA256 + ":" + item.SHA256
					if item.SHA256 < d.SHA256 {
						key = item.SHA256 + ":" + d.SHA256
					}

					if _, ok := done[key]; ok {
						continue
					}

					if item.FullPath != d.FullPath {
						// only here if not looking at same disk
						l.Logf(":: Checking overlapping files %s", item.Filename)
						dSame := d.CompareFiles(&item)

						item.MatchFactor = dSame

						if dSame >= t {
							matchlist[key] = [2]*Disk{&d, &item}
						}
					}

					done[key] = true

				}

			}

			fmt.Print("\r")
			lastPc = pc
		}
	}

	return matchlist
}

const ingestWorkers = 4
const processWorkers = 6

func exists(path string) bool {

	_, err := os.Stat(path)
	if err != nil {
		return false
	}
	return true

}
