package disk

import (
	"bytes"
	"regexp"
	"strings"
)

const RDOS_CATALOG_TRACK = 0x01
const RDOS_CATALOG_LENGTH = 0xB
const RDOS_ENTRY_LENGTH = 0x20
const RDOS_NAME_LENGTH = 0x18

var RDOS_SIGNATURE = []byte{
	byte('R' + 0x80),
	byte('D' + 0x80),
	byte('O' + 0x80),
	byte('S' + 0x80),
	byte(' ' + 0x80),
}

var RDOS_SIGNATURE_32 = []byte{
	byte('R' + 0x80),
	byte('D' + 0x80),
	byte('O' + 0x80),
	byte('S' + 0x80),
	byte(' ' + 0x80),
	byte('2' + 0x80),
}

var RDOS_SIGNATURE_33 = []byte{
	byte('R' + 0x80),
	byte('D' + 0x80),
	byte('O' + 0x80),
	byte('S' + 0x80),
	byte(' ' + 0x80),
	byte('3' + 0x80),
}

type RDOSFormatSpec struct {
	SectorStride  int
	SectorMax     int
	CatalogTrack  int
	CatalogSector int
	Ordering      SectorOrder
}

type RDOSFormat int

const (
	RDOS_Unknown RDOSFormat = iota
	RDOS_3
	RDOS_32
	RDOS_33
)

func (f RDOSFormat) String() string {

	switch f {
	case RDOS_3:
		return "RDOS3"
	case RDOS_32:
		return "RDOS32"
	case RDOS_33:
		return "RDOS33"
	}

	return "Unknown"

}

func (f RDOSFormat) Spec() *RDOSFormatSpec {

	switch f {
	case RDOS_32:
		return &RDOSFormatSpec{
			SectorStride:  13,
			SectorMax:     13,
			CatalogTrack:  1,
			CatalogSector: 0,
			Ordering:      SectorOrderDOS33,
		}
	case RDOS_3:
		return &RDOSFormatSpec{
			SectorStride:  16,
			SectorMax:     13,
			CatalogTrack:  0,
			CatalogSector: 1,
			Ordering:      SectorOrderDOS33,
		}
	case RDOS_33:
		return &RDOSFormatSpec{
			SectorStride:  16,
			SectorMax:     16,
			CatalogTrack:  1,
			CatalogSector: 12,
			Ordering:      SectorOrderProDOS,
		}
	}
	return nil

}

func (dsk *DSKWrapper) IsRDOS() (bool, RDOSFormat) {

	// It needs to be either 140K or 113K
	if len(dsk.Data) != STD_DISK_BYTES && len(dsk.Data) != STD_DISK_BYTES_OLD {
		return false, RDOS_Unknown
	}

	sectorStride := (len(dsk.Data) / STD_TRACKS_PER_DISK) / 256

	idbytes := dsk.Data[sectorStride*256 : sectorStride*256+6]

	if bytes.Compare(idbytes, RDOS_SIGNATURE_32) == 0 && sectorStride == 13 {
		return true, RDOS_32
	}

	if bytes.Compare(idbytes, RDOS_SIGNATURE_32) == 0 && sectorStride == 16 {
		return true, RDOS_3
	}

	if bytes.Compare(idbytes, RDOS_SIGNATURE_33) == 0 && sectorStride == 16 {
		return true, RDOS_33
	}

	return false, RDOS_Unknown

}

type RDOSFileDescriptor struct {
	data [RDOS_ENTRY_LENGTH]byte
}

func (fd *RDOSFileDescriptor) SetData(in []byte) {
	for i, b := range in {
		if i < RDOS_ENTRY_LENGTH {
			fd.data[i] = b
		}
	}
}

func (fd *RDOSFileDescriptor) IsDeleted() bool {

	return fd.data[24] == 0xa0 || fd.data[0] == 0x80

}

func (fd *RDOSFileDescriptor) IsUnused() bool {

	return fd.data[24] == 0x00

}

func (fd *RDOSFileDescriptor) IsLocked() bool {
	return true
}

type RDOSFileType int

const (
	FileType_RDOS_Unknown RDOSFileType = iota
	FileType_RDOS_AppleSoft
	FileType_RDOS_Binary
	FileType_RDOS_Text
)

var RDOSTypeMap = map[RDOSFileType][2]string{
	FileType_RDOS_Unknown:   [2]string{"UNK", "Unknown"},
	FileType_RDOS_AppleSoft: [2]string{"APP", "Applesoft Basic Program"},
	FileType_RDOS_Binary:    [2]string{"BIN", "Binary File"},
	FileType_RDOS_Text:      [2]string{"TXT", "ASCII Text"},
}

func (ft RDOSFileType) String() string {
	info, ok := RDOSTypeMap[ft]
	if ok {
		return info[1]
	}
	return "Unknown"
}

func (ft RDOSFileType) Ext() string {
	info, ok := RDOSTypeMap[ft]
	if ok {
		return info[0]
	}
	return "UNK"
}

func RDOSFileTypeFromExt(ext string) RDOSFileType {
	for ft, info := range RDOSTypeMap {
		if strings.ToUpper(ext) == info[0] {
			return ft
		}
	}
	return 0x00
}

func (fd *RDOSFileDescriptor) Type() RDOSFileType {

	switch rune(fd.data[24]) {
	case 'A' + 0x80:
		return FileType_RDOS_AppleSoft
	case 'B' + 0x80:
		return FileType_RDOS_Binary
	case 'T' + 0x80:
		return FileType_RDOS_Text
	}

	return FileType_RDOS_Unknown

}

func (fd *RDOSFileDescriptor) Name() string {

	str := ""
	for i := 0; i < RDOS_NAME_LENGTH; i++ {

		ch := rune(fd.data[i] & 127)
		if ch == 0 {
			break
		}
		str += string(ch)

	}

	str = strings.TrimRight(str, " ")
	switch fd.Type() {
	case FileType_RDOS_AppleSoft:
		str += ".a"
	case FileType_RDOS_Binary:
		str += ".s"
	case FileType_RDOS_Text:
		str += ".t"
	}

	return str

}

func (fd *RDOSFileDescriptor) NameUnadorned() string {

	str := ""
	for i := 0; i < RDOS_NAME_LENGTH; i++ {

		ch := rune(fd.data[i] & 127)
		if ch == 0 {
			break
		}
		str += string(ch)

	}

	return str

}

func (fd RDOSFileDescriptor) NumSectors() int {
	return int(fd.data[25])
}

func (fd RDOSFileDescriptor) LoadAddress() int {
	return int(fd.data[26]) + 256*int(fd.data[27])
}

func (fd RDOSFileDescriptor) StartSector() int {
	return int(fd.data[30]) + 256*int(fd.data[31])
}

func (fd RDOSFileDescriptor) Length() int {
	return int(fd.data[28]) + 256*int(fd.data[29])
}

func (dsk *DSKWrapper) RDOSGetCatalog(pattern string) ([]*RDOSFileDescriptor, error) {

	pattern = strings.Replace(pattern, ".", "[.]", -1)
	pattern = strings.Replace(pattern, "*", ".*", -1)
	pattern = strings.Replace(pattern, "?", ".", -1)

	rx := regexp.MustCompile("(?i)" + pattern)

	var files = make([]*RDOSFileDescriptor, 0)

	d := make([]byte, 0)

	for s := 0; s < RDOS_CATALOG_LENGTH; s++ {
		dsk.SetTrack(1)
		dsk.SetSector(s)
		chunk := dsk.Read()
		d = append(d, chunk...)
	}

	var dirPtr int
	for i := 0; i < RDOS_CATALOG_LENGTH*RDOS_ENTRY_LENGTH; i++ {
		entry := &RDOSFileDescriptor{}
		entry.SetData(d[dirPtr : dirPtr+RDOS_ENTRY_LENGTH])

		dirPtr += RDOS_ENTRY_LENGTH

		if entry.IsUnused() {
			break
		}

		if !entry.IsDeleted() && rx.MatchString(entry.NameUnadorned()) {
			files = append(files, entry)
		}

	}

	return files, nil

}

func (dsk *DSKWrapper) RDOSUsedBitmap() ([]bool, error) {

	spt := dsk.RDOSFormat.Spec().SectorMax
	activeSectors := spt * 35

	used := make([]bool, activeSectors)

	files, err := dsk.RDOSGetCatalog("*")
	if err != nil {
		return used, err
	}

	for _, file := range files {

		length := file.NumSectors()
		start := file.StartSector()
		if start+length > activeSectors {
			continue // file is bad
		}

		for block := start; block < start+length; block++ {
			used[block] = true
		}

	}

	return used, nil

}

func (dsk *DSKWrapper) RDOSReadFile(file *RDOSFileDescriptor) ([]byte, error) {

	spt := dsk.RDOSFormat.Spec().SectorMax
	activeSectors := spt * 35

	length := file.NumSectors()
	start := file.StartSector()

	// If file is damaged return nothing
	if start+length > activeSectors {
		return []byte(nil), nil
	}

	block := start
	data := make([]byte, 0)
	for block < start+length && len(data) < file.Length() {

		track := block / spt
		sector := block % spt

		dsk.SetTrack(track)
		dsk.SetSector(sector)

		chunk := dsk.Read()
		needed := file.Length() - len(data)
		if needed >= 256 {
			data = append(data, chunk...)
		} else {
			data = append(data, chunk[:needed]...)
		}

		block++

	}

	return data, nil

}
