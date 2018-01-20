package disk

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type FileType byte

const (
	FileTypeTXT FileType = 0x00
	FileTypeINT FileType = 0x01
	FileTypeAPP FileType = 0x02
	FileTypeBIN FileType = 0x04
	FileTypeS   FileType = 0x08
	FileTypeREL FileType = 0x10
	FileTypeA   FileType = 0x20
	FileTypeB   FileType = 0x40
)

var AppleDOSTypeMap = map[FileType][2]string{
	0x00: [2]string{"TXT", "ASCII Text"},
	0x01: [2]string{"INT", "Integer Basic Program"},
	0x02: [2]string{"BAS", "Applesoft Basic Program"},
	0x04: [2]string{"BIN", "Binary File"},
	0x08: [2]string{"S", "S File Type"},
	0x10: [2]string{"REL", "Relocatable Object Code"},
	0x20: [2]string{"A", "A File Type"},
	0x40: [2]string{"B", "B File Type"},
}

func (ft FileType) String() string {

	info, ok := AppleDOSTypeMap[ft]
	if ok {
		return info[1]
	}

	return "Unknown"
}

func AppleDOSFileTypeFromExt(ext string) FileType {
	for ft, info := range AppleDOSTypeMap {
		if strings.ToUpper(ext) == info[0] {
			return ft
		}
	}
	return 0x04
}

func (ft FileType) Ext() string {

	info, ok := AppleDOSTypeMap[ft]
	if ok {
		return info[0]
	}

	return "BIN"
}

type FileDescriptor struct {
	Data              []byte
	trackid, sectorid int
	sectoroffset      int
}

func (fd *FileDescriptor) SetData(data []byte, t, s, o int) {

	fd.trackid = t
	fd.sectorid = s
	fd.sectoroffset = o

	if fd.Data == nil && len(data) == 35 {
		fd.Data = data
	}

	for i, v := range data {
		fd.Data[i] = v
	}
}

func (fd *FileDescriptor) Publish(dsk *DSKWrapper) error {

	err := dsk.Seek(fd.trackid, fd.sectorid)
	if err != nil {
		return err
	}
	block := dsk.Read()

	for i, v := range fd.Data {
		block[fd.sectoroffset+i] = v
	}

	dsk.Write(block)

	return nil

}

func (fd *FileDescriptor) IsUnused() bool {
	return fd.Data[0] == 0xff || fd.Type().String() == "Unknown" || fd.TotalSectors() == 0
}

func (fd *FileDescriptor) GetTrackSectorListStart() (int, int) {
	return int(fd.Data[0]), int(fd.Data[1])
}

func (fd *FileDescriptor) IsLocked() bool {
	return (fd.Data[2]&0x80 != 0)
}

func (fd *FileDescriptor) SetLocked(b bool) {
	fd.Data[2] = fd.Data[2] & 0x7f
	if b {
		fd.Data[2] = fd.Data[2] | 0x80
	}
}

func (fd *FileDescriptor) Type() FileType {
	return FileType(fd.Data[2] & 0x7f)
}

func (fd *FileDescriptor) SetType(t FileType) {
	fd.Data[0x02] = byte(t)
}

func AsciiToPoke(b byte) byte {
	if b < 32 || b > 127 {
		b = 32
	}
	return b | 128
}

func (fd *FileDescriptor) SetName(s string) {
	maxlen := len(s)
	if maxlen > 30 {
		maxlen = 30
	}
	for i := 0; i < 30; i++ {
		fd.Data[0x03+i] = 0xa0
	}
	for i, b := range []byte(s) {
		if i >= maxlen {
			break
		}
		fd.Data[0x03+i] = AsciiToPoke(b)
	}
}

func (fd *FileDescriptor) Name() string {
	r := fd.Data[0x03:0x21]
	s := ""
	for _, v := range r {
		ch := PokeToAscii(uint(v), false)
		s = s + string(ch)
	}

	s = strings.ToLower(strings.Trim(s, " "))

	switch fd.Type() {
	case FileTypeAPP:
		s += ".a"
	case FileTypeINT:
		s += ".i"
	case FileTypeBIN:
		s += ".s"
	case FileTypeTXT:
		s += ".t"
	}

	return s
}

func (fd *FileDescriptor) NameUnadorned() string {
	r := fd.Data[0x03:0x21]
	s := ""
	for _, v := range r {
		ch := PokeToAscii(uint(v), false)
		s = s + string(ch)
	}

	s = strings.ToLower(strings.Trim(s, " "))

	return s
}

func (fd *FileDescriptor) NameBytes() []byte {
	return fd.Data[0x03:0x21]
}

func (fd *FileDescriptor) NameOK() bool {
	for _, v := range fd.NameBytes() {
		if v < 32 {
			return false
		}
	}

	return true
}

func (fd *FileDescriptor) TotalSectors() int {
	return int(fd.Data[0x21]) + 256*int(fd.Data[0x22])
}

func (fd *FileDescriptor) SetTotalSectors(v int) {
	fd.Data[0x21] = byte(v & 0xff)
	fd.Data[0x22] = byte(v / 0x100)
}

func (fd *FileDescriptor) SetTrackSectorListStart(t, s int) {
	fd.Data[0] = byte(t)
	fd.Data[1] = byte(s)
}

type VTOC struct {
	Data [256]byte
	t, s int
}

func (fd *VTOC) SetData(data []byte, t, s int) {
	fd.t, fd.s = t, s
	for i, v := range data {
		fd.Data[i] = v
	}
}

func (fd *VTOC) GetCatalogStart() (int, int) {
	return int(fd.Data[1]), int(fd.Data[2])
}

func (fd *VTOC) GetDOSVersion() byte {
	return fd.Data[3]
}

func (fd *VTOC) GetVolumeID() byte {
	return fd.Data[6]
}

func (fd *VTOC) GetMaxTSPairsPerSector() int {
	return int(fd.Data[0x27])
}

func (fd *VTOC) GetTracks() int {
	return int(fd.Data[0x34])
}

func (fd *VTOC) GetSectors() int {
	return int(fd.Data[0x35])
}

func (fd *VTOC) GetTrackOrder() int {
	return int(fd.Data[0x31])
}

func (fd *VTOC) BytesPerSector() int {
	return int(fd.Data[0x36]) + 256*int(fd.Data[0x37])
}

func (fd *VTOC) IsTSFree(t, s int) bool {
	offset := 0x38 + t*4
	if s < 8 {
		offset++
	}
	bitmask := byte(1 << uint(s&0x7))

	return (fd.Data[offset]&bitmask != 0)
}

// SetTSFree marks a T/S free or not
func (fd *VTOC) SetTSFree(t, s int, b bool) {
	offset := 0x38 + t*4
	if s < 8 {
		offset++
	}
	bitmask := byte(1 << uint(s&0x7))
	clrmask := 0xff ^ bitmask

	v := fd.Data[offset]
	if b {
		v |= bitmask
	} else {
		v &= clrmask
	}

	fd.Data[offset] = v
}

func (fd *VTOC) Publish(dsk *DSKWrapper) error {
	err := dsk.Seek(fd.t, fd.s)
	if err != nil {
		return err
	}

	dsk.Write(fd.Data[:])

	return nil
}

func (fd *VTOC) DumpMap() {

	fmt.Printf("Disk has %d tracks and %d sectors per track, %d bytes per sector (ordering %d)...\n", fd.GetTracks(), fd.GetSectors(), fd.BytesPerSector(), fd.GetTrackOrder())
	fmt.Printf("Volume ID is %d\n", fd.GetVolumeID())
	ct, cs := fd.GetCatalogStart()
	fmt.Printf("Catalog starts at T%d, S%d\n", ct, cs)

	tcount := fd.GetTracks()
	scount := fd.GetSectors()

	for t := 0; t < tcount; t++ {
		fmt.Printf("TRACK %.2x: |", t)
		for s := 0; s < scount; s++ {
			if fd.IsTSFree(t, s) {
				fmt.Print(".")
			} else {
				fmt.Print("X")
			}
		}
		fmt.Println("|")
	}

}

func (dsk *DSKWrapper) IsAppleDOS() (bool, DiskFormat, SectorOrder) {

	oldFormat := dsk.Format
	oldLayout := dsk.Layout

	defer func() {
		dsk.Format = oldFormat
		dsk.Layout = oldLayout
	}()

	if len(dsk.Data) == STD_DISK_BYTES {

		layouts := []SectorOrder{SectorOrderDOS33, SectorOrderDOS33Alt, SectorOrderProDOS, SectorOrderProDOSLinear}

		for _, l := range layouts {

			dsk.Layout = l

			vtoc, err := dsk.AppleDOSGetVTOC()
			if err != nil {
				continue
			}

			if vtoc.GetTracks() != 35 || vtoc.GetSectors() != 16 {
				continue
			}

			_, files, err := dsk.AppleDOSGetCatalog("*")
			if err != nil {
				continue
			}

			if len(files) > 0 {
				return true, GetDiskFormat(DF_DOS_SECTORS_16), l
			}

		}

	} else if len(dsk.Data) == STD_DISK_BYTES_OLD {

		layouts := []SectorOrder{SectorOrderDOS33, SectorOrderDOS33Alt, SectorOrderProDOS, SectorOrderProDOSLinear}

		dsk.Format = GetDiskFormat(DF_DOS_SECTORS_13)

		for _, l := range layouts {
			dsk.Layout = l

			vtoc, err := dsk.AppleDOSGetVTOC()
			if err != nil {
				continue
			}

			if vtoc.GetTracks() != 35 || vtoc.GetSectors() != 13 {
				continue
			}

			_, files, err := dsk.AppleDOSGetCatalog("*")
			if err != nil {
				continue
			}

			if len(files) > 0 {
				return true, GetDiskFormat(DF_DOS_SECTORS_13), l
			}

		}

	}

	return false, oldFormat, oldLayout

}

func (d *DSKWrapper) AppleDOSReadFileRaw(fd FileDescriptor) (int, int, []byte, error) {

	data, e := d.AppleDOSReadFileSectors(fd, -1)

	if e != nil || len(data) == 0 {
		return 0, 0, data, e
	}

	switch fd.Type() {
	case FileTypeINT:
		l := int(data[0]) + 256*int(data[1])
		if l+2 > len(data) {
			l = len(data) - 2
		}
		return l, 0x801, data[2 : 2+l], nil
	case FileTypeAPP:
		l := int(data[0]) + 256*int(data[1])
		if l+2 > len(data) {
			l = len(data) - 2
		}
		return l, 0x801, data[2 : 2+l], nil
	case FileTypeTXT:
		return len(data), 0x0000, data, nil
	case FileTypeBIN:
		addr := int(data[0]) + 256*int(data[1])
		l := int(data[2]) + 256*int(data[3])
		if l+4 > len(data) {
			l = len(data) - 4
		}
		//fmt.Printf("%x, %x, %x\n", l, addr, len(data))
		return l, addr, data[4 : 4+l], nil
	default:
		l := int(data[0]) + 256*int(data[1])
		if l+2 > len(data) {
			l = len(data) - 2
		}
		return l, 0, data[2 : 2+l], nil
	}

}

func (d *DSKWrapper) AppleDOSGetVTOC() (*VTOC, error) {
	e := d.Seek(17, 0)
	if e != nil {
		return nil, e
	}
	data := d.Read()

	vtoc := &VTOC{}
	vtoc.SetData(data, 17, 0)
	return vtoc, nil
}

func (dsk *DSKWrapper) AppleDOSUsedBitmap() ([]bool, error) {

	var out []bool = make([]bool, dsk.Format.TPD()*dsk.Format.SPT())

	_, files, err := dsk.AppleDOSGetCatalog("*")
	if err != nil {
		return out, err
	}

	for _, f := range files {
		tslist, err := dsk.AppleDOSGetFileSectors(f, 0)
		if err == nil {
			for _, pair := range tslist {
				track := pair[0]
				sector := pair[1]
				out[track*dsk.Format.SPT()+sector] = true
			}
		}
	}

	return out, nil

}

func (d *DSKWrapper) AppleDOSGetCatalog(pattern string) (*VTOC, []FileDescriptor, error) {

	var files []FileDescriptor
	var e error
	var vtoc *VTOC

	vtoc, e = d.AppleDOSGetVTOC()
	if e != nil {
		return vtoc, files, e
	}

	count := 0
	ct, cs := vtoc.GetCatalogStart()

	e = d.Seek(ct, cs)
	if e != nil {
		return vtoc, files, e
	}

	data := d.Read()

	var re *regexp.Regexp
	if pattern != "" {
		patterntmp := strings.Replace(pattern, ".", "[.]", -1)
		patterntmp = strings.Replace(patterntmp, "*", ".*", -1)
		patterntmp = "(?i)^" + patterntmp + "$"
		re = regexp.MustCompile(patterntmp)
	}

	for e == nil && count < 105 {
		slot := count % 7
		pos := 0x0b + 35*slot

		fd := FileDescriptor{}
		fd.SetData(data[pos:pos+35], ct, cs, pos)

		var skipname bool = false
		if re != nil {
			skipname = !re.MatchString(fd.Name())
		}

		if fd.Data[0] != 0xff && fd.Data[0] != 0x00 && fd.Type().String() != "Unknown" && !skipname {
			files = append(files, fd)
		}
		count++
		if count%7 == 0 {
			// move to next catalog sector
			ct = int(data[1])
			cs = int(data[2])
			if ct == 0 {
				return vtoc, files, nil
			}
			e = d.Seek(ct, cs)
			if e != nil {
				return vtoc, files, e
			}
			data = d.Read()
		}
	}

	return vtoc, files, nil
}

func (d *DSKWrapper) AppleDOSGetFileSectors(fd FileDescriptor, maxblocks int) ([][2]int, error) {
	var e error
	var data []byte
	tl, sl := fd.GetTrackSectorListStart()

	// var tracks []int
	// var sectors []int

	var tslist [][2]int

	var tsmap = make(map[int]int)

	for e == nil && (tl != 0 || sl != 0) {
		// Get TS List
		e = d.Seek(tl, sl)
		if e != nil {
			return tslist, e
		}
		data = d.Read()

		//fmt.Printf("DEBUG: T/S List follows from T%d, S%d:\n", tl, sl)
		//Dump(data)

		ptr := 0x0c
		for ptr < 0x100 {
			// check entry
			t, s := int(data[ptr]), int(data[ptr+1])

			if t == 0 && s == 0 || t >= d.Format.TPD() || s >= d.Format.SPT() {
				//fmt.Println("BREAK ptr =", ptr, len(tracks))
				break
			}

			//fmt.Printf("File block at T%d, S%d\n", t, s)

			// tracks = append(tracks, t)
			// sectors = append(sectors, s)

			tslist = append(tslist, [2]int{t, s})

			// next entry
			ptr += 2
		}

		// get next TS List block
		ntl, nsl := int(data[1]), int(data[2])
		if _, ex := tsmap[100*ntl+nsl]; ex {
			//fmt.Printf("circular ts list")
			break
		}

		tl, sl = ntl, nsl

		tsmap[100*tl+sl] = 1

		//fmt.Printf("Next Track Sector list is at T%d, S%d (%d)\n", tl, sl, len(tracks))

	}

	return tslist, nil
}

func (d *DSKWrapper) AppleDOSReadFileSectors(fd FileDescriptor, maxblocks int) ([]byte, error) {
	var e error
	var data []byte
	var file []byte
	tl, sl := fd.GetTrackSectorListStart()

	var tracks []int
	var sectors []int

	var tsmap = make(map[int]int)

	for e == nil && (tl != 0 || sl != 0) {
		// Get TS List
		e = d.Seek(tl, sl)
		if e != nil {
			return file, e
		}
		data = d.Read()

		//fmt.Printf("DEBUG: T/S List follows from T%d, S%d:\n", tl, sl)
		//Dump(data)

		ptr := 0x0c
		for ptr < 0x100 {
			// check entry
			t, s := int(data[ptr]), int(data[ptr+1])

			if t == 0 && s == 0 || t >= d.Format.TPD() || s >= d.Format.SPT() {
				//fmt.Println("BREAK ptr =", ptr, len(tracks))
				break
			}

			//fmt.Printf("File block at T%d, S%d\n", t, s)

			tracks = append(tracks, t)
			sectors = append(sectors, s)

			// next entry
			ptr += 2
		}

		// get next TS List block
		ntl, nsl := int(data[1]), int(data[2])
		if _, ex := tsmap[100*ntl+nsl]; ex {
			//fmt.Printf("circular ts list")
			break
		}

		tl, sl = ntl, nsl

		tsmap[100*tl+sl] = 1

		//fmt.Printf("Next Track Sector list is at T%d, S%d (%d)\n", tl, sl, len(tracks))

	}

	// Here got T/S list
	//fmt.Println("READING FILE")
	blocksread := 0
	for i, t := range tracks {
		s := sectors[i]

		//fmt.Printf("TS Fetch #%d: Track %d, %d\n", i, t, s)

		e = d.Seek(t, s)
		if e != nil {
			return file, e
		}
		c := d.Read()

		//Dump(c)

		file = append(file, c...)
		blocksread++

		if maxblocks != -1 && blocksread >= maxblocks {
			break
		}
	}

	return file, nil
}

func (d *DSKWrapper) AppleDOSGetTSListSectors(fd FileDescriptor, maxblocks int) ([][2]int, error) {
	var e error
	var data []byte

	tl, sl := fd.GetTrackSectorListStart()

	var tslist [][2]int

	var tsmap = make(map[int]int)

	for e == nil && (tl != 0 || sl != 0) {
		// Get TS List
		e = d.Seek(tl, sl)
		if e != nil {
			return tslist, e
		}
		data = d.Read()

		tslist = append(tslist, [2]int{tl, sl})

		// get next TS List block
		ntl, nsl := int(data[1]), int(data[2])
		if _, ex := tsmap[100*ntl+nsl]; ex {
			break
		}

		tl, sl = ntl, nsl

		tsmap[100*tl+sl] = 1

	}

	return tslist, nil
}

func (d *DSKWrapper) AppleDOSReadFile(fd FileDescriptor) (int, int, []byte, error) {

	data, e := d.AppleDOSReadFileSectors(fd, -1)

	if e != nil {
		return 0, 0, data, e
	}

	switch fd.Type() {
	case FileTypeINT:
		l := int(data[0]) + 256*int(data[1])
		return l, 0x801, IntegerDetoks(data[2 : 2+l]), nil
	case FileTypeAPP:
		l := int(data[0]) + 256*int(data[1])
		return l, 0x801, ApplesoftDetoks(data[2 : 2+l]), nil
	case FileTypeTXT:
		return len(data), 0x0000, data, nil
	case FileTypeBIN:
		addr := int(data[0]) + 256*int(data[1])
		l := int(data[2]) + 256*int(data[3])
		//fmt.Printf("%x, %x, %x\n", l, addr, len(data))
		return l, addr, data[4 : 4+l], nil
	default:
		l := int(data[0]) + 256*int(data[1])
		return l, 0, data[2 : 2+l], nil
	}

}

// AppleDOSGetFreeSectors tries to find free sectors for certain size file...
// Remember, we need space for the T/S list as well...
func (dsk *DSKWrapper) AppleDOSGetFreeSectors(size int) ([][2]int, [][2]int, error) {

	needed := make([][2]int, 0)
	vtoc, err := dsk.AppleDOSGetVTOC()
	if err != nil {
		return nil, nil, err
	}

	catTrack, _ := vtoc.GetCatalogStart()

	// needed:
	// size/256 + 1 for data
	// 1 for T/S list
	dataBlocks := (size / 256) + 1
	tsListBlocks := (dataBlocks / vtoc.GetMaxTSPairsPerSector()) + 1
	totalBlocks := tsListBlocks + dataBlocks

	for t := dsk.Format.TPD() - 1; t >= 0; t-- {

		if t == catTrack {
			continue // skip catalog track
		}

		for s := dsk.Format.SPT() - 1; s >= 0; s-- {

			if len(needed) >= totalBlocks {
				break
			}

			if vtoc.IsTSFree(t, s) {
				needed = append(needed, [2]int{t, s})
			}
		}

	}

	if len(needed) >= totalBlocks {
		return needed[:tsListBlocks], needed[tsListBlocks:], nil
	}

	return nil, nil, errors.New("Insufficent space")

}

func (d *DSKWrapper) AppleDOSNextFreeCatalogEntry(name string) (*FileDescriptor, error) {

	var e error
	var vtoc *VTOC

	vtoc, e = d.AppleDOSGetVTOC()
	if e != nil {
		return nil, e
	}

	count := 0
	ct, cs := vtoc.GetCatalogStart()

	e = d.Seek(ct, cs)
	if e != nil {
		return nil, e
	}

	data := d.Read()

	for e == nil && count < 105 {
		slot := count % 7
		pos := 0x0b + 35*slot

		fd := FileDescriptor{}
		fd.SetData(data[pos:pos+35], ct, cs, pos)

		if fd.IsUnused() {
			return &fd, nil
		} else if name != "" && strings.ToLower(fd.NameUnadorned()) == strings.ToLower(name) {
			return &fd, nil
		}
		count++
		if count%7 == 0 {
			// move to next catalog sector
			ct = int(data[1])
			cs = int(data[2])
			if ct == 0 {
				return nil, nil
			}
			e = d.Seek(ct, cs)
			if e != nil {
				return nil, e
			}
			data = d.Read()
		}
	}

	return nil, errors.New("No free entry")
}

func (d *DSKWrapper) AppleDOSNamedCatalogEntry(name string) (*FileDescriptor, error) {

	var e error
	var vtoc *VTOC

	vtoc, e = d.AppleDOSGetVTOC()
	if e != nil {
		return nil, e
	}

	count := 0
	ct, cs := vtoc.GetCatalogStart()

	e = d.Seek(ct, cs)
	if e != nil {
		return nil, e
	}

	data := d.Read()

	for e == nil && count < 105 {
		slot := count % 7
		pos := 0x0b + 35*slot

		fd := FileDescriptor{}
		fd.SetData(data[pos:pos+35], ct, cs, pos)

		//fmt.Printf("FILE NAME CHECK [%s] vs [%s]\n", strings.ToLower(fd.NameUnadorned()), strings.ToLower(name))

		if name != "" && strings.ToLower(fd.NameUnadorned()) == strings.ToLower(name) {
			return &fd, nil
		}
		count++
		if count%7 == 0 {
			// move to next catalog sector
			ct = int(data[1])
			cs = int(data[2])
			if ct == 0 {
				return nil, errors.New("Not found")
			}
			e = d.Seek(ct, cs)
			if e != nil {
				return nil, e
			}
			data = d.Read()
		}
	}

	return nil, errors.New("Not found")
}

func (dsk *DSKWrapper) AppleDOSWriteFile(name string, kind FileType, data []byte, loadAddr int) error {

	name = strings.ToUpper(name)

	vtoc, err := dsk.AppleDOSGetVTOC()
	if err != nil {
		return err
	}

	if kind != FileTypeTXT {
		header := []byte{byte(loadAddr % 256), byte(loadAddr / 256)}
		data = append(header, data...)
	}

	// 1st: check we have sufficient space...
	tsBlocks, dataBlocks, err := dsk.AppleDOSGetFreeSectors(len(data))
	if err != nil {
		return err
	}

	fd, err := dsk.AppleDOSNamedCatalogEntry(name)
	//fmt.Println("FD=", fd)
	if err == nil {
		if kind != fd.Type() {
			return errors.New("File type mismatch")
		} else {
			// need to delete this file...
			err = dsk.AppleDOSDeleteFile(name)
			if err != nil {
				return err
			}
		}
	} else {
		fd, err = dsk.AppleDOSNextFreeCatalogEntry(name)
		if err != nil {
			return err
		}
	}

	// 2nd: check we can get a free catalog entry

	// 3rd: Write the datablocks
	var block int = 0
	for len(data) > 0 {

		max := STD_BYTES_PER_SECTOR
		if len(data) < STD_BYTES_PER_SECTOR {
			max = len(data)
		}
		chunk := data[:max]
		// Pad final sector with 0x00 bytes
		for len(chunk) < STD_BYTES_PER_SECTOR {
			chunk = append(chunk, 0x00)
		}
		data = data[max:]

		pair := dataBlocks[block]

		track, sector := pair[0], pair[1]

		err = dsk.Seek(track, sector)
		if err != nil {
			return err
		}
		dsk.Write(chunk)

		block++

	}

	// 4th: Write the T/S List
	offset := 0
	for blockIdx, block := range tsBlocks {
		listTrack, listSector := block[0], block[1]
		nextTrack, nextSector := 0, 0
		if blockIdx < len(tsBlocks)-1 {
			nextTrack, nextSector = tsBlocks[blockIdx+1][0], tsBlocks[blockIdx+1][1]
		}

		buffer := make([]byte, STD_BYTES_PER_SECTOR)

		// header
		buffer[0x01] = byte(nextTrack)
		buffer[0x02] = byte(nextSector)
		buffer[0x05] = byte(offset & 0xff)
		buffer[0x06] = byte(offset / 0x100)

		//
		count := vtoc.GetMaxTSPairsPerSector()
		if offset+count >= len(dataBlocks) {
			count = len(dataBlocks) - offset
		}

		for i := 0; i < count; i++ {
			pos := 0x0c + i*2
			buffer[pos+0x00] = byte(dataBlocks[offset+i][0])
			buffer[pos+0x01] = byte(dataBlocks[offset+i][1])
			vtoc.SetTSFree(dataBlocks[offset+i][0], dataBlocks[offset+i][1], false)
		}

		// Write the sector
		err = dsk.Seek(listTrack, listSector)
		if err != nil {
			return err
		}
		dsk.Write(buffer)
		vtoc.SetTSFree(listTrack, listSector, false)
	}

	err = vtoc.Publish(dsk)
	if err != nil {
		return err
	}

	// 5th and finally: Let's make that catalog entry
	fd.SetName(name)
	fd.SetTrackSectorListStart(tsBlocks[0][0], tsBlocks[0][1])
	fd.SetType(kind)
	fd.SetTotalSectors(len(dataBlocks))

	return nil

}

func (d *DSKWrapper) AppleDOSRemoveFile(fd *FileDescriptor) error {

	vtoc, err := d.AppleDOSGetVTOC()
	if err != nil {
		return err
	}

	if fd.IsUnused() {
		return errors.New("File does not exist")
	}

	tsBlocks, e := d.AppleDOSGetTSListSectors(*fd, -1)
	if e != nil {
		return e
	}

	dataBlocks, e := d.AppleDOSGetFileSectors(*fd, -1)
	if e != nil {
		return e
	}

	for _, pair := range dataBlocks {
		vtoc.SetTSFree(pair[0], pair[1], true)
	}

	for _, pair := range tsBlocks {
		vtoc.SetTSFree(pair[0], pair[1], true)
	}

	fd.Data[0x00] = 0xff
	fd.SetName("")
	return fd.Publish(d)

}

func (dsk *DSKWrapper) AppleDOSDeleteFile(name string) error {

	vtoc, err := dsk.AppleDOSGetVTOC()
	if err != nil {
		return err
	}

	// We cheat here a bit and use the get first free entry call with
	// autogrow turned off.
	fd, err := dsk.AppleDOSNamedCatalogEntry(name)
	if err != nil {
		return err
	}

	if fd.IsUnused() {
		return errors.New("Not found")
	}

	// At this stage we have a match so get blocks to remove
	tsBlocks, e := dsk.AppleDOSGetTSListSectors(*fd, -1)
	if e != nil {
		return e
	}

	dataBlocks, e := dsk.AppleDOSGetFileSectors(*fd, -1)
	if e != nil {
		return e
	}

	for _, pair := range dataBlocks {
		vtoc.SetTSFree(pair[0], pair[1], true)
	}

	for _, pair := range tsBlocks {
		vtoc.SetTSFree(pair[0], pair[1], true)
	}

	err = vtoc.Publish(dsk)
	if err != nil {
		return err
	}

	fd.Data[0x00] = 0xff
	fd.SetName("")
	return fd.Publish(dsk)

}

func (dsk *DSKWrapper) AppleDOSSetLocked(name string, lock bool) error {

	// We cheat here a bit and use the get first free entry call with
	// autogrow turned off.
	fd, err := dsk.AppleDOSNamedCatalogEntry(name)
	if err != nil {
		return err
	}

	if fd.IsUnused() {
		return errors.New("Not found")
	}

	fd.SetLocked(lock)
	return fd.Publish(dsk)

}

func (dsk *DSKWrapper) AppleDOSRenameFile(name, newname string) error {

	fd, err := dsk.AppleDOSNamedCatalogEntry(name)
	if err != nil {
		return err
	}

	_, err = dsk.AppleDOSNamedCatalogEntry(newname)
	if err == nil {
		return errors.New("New name already exists")
	}

	// can rename here
	fd.SetName(newname)
	return fd.Publish(dsk)

}
