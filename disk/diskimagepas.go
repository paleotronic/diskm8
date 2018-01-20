package disk

import (
	"errors"
	"regexp"
	"strings"
)

const PASCAL_BLOCK_SIZE = 512
const PASCAL_VOLUME_BLOCK = 2
const PASCAL_MAX_VOLUME_NAME = 7
const PASCAL_DIRECTORY_ENTRY_LENGTH = 26
const PASCAL_OVERSIZE_DIR = 32

func (dsk *DSKWrapper) IsPascal() (bool, string) {

	dsk.Format = GetDiskFormat(DF_PRODOS)

	data, err := dsk.PRODOSGetBlock(PASCAL_VOLUME_BLOCK)
	if err != nil {
		return false, ""
	}

	if !(data[0x00] == 0 && data[0x01] == 0) ||
		!(data[0x04] == 0 && data[0x05] == 0) ||
		!(data[0x06] > 0 && data[0x06] <= PASCAL_MAX_VOLUME_NAME) {
		return false, ""
	}

	l := int(data[0x06])
	name := data[0x07 : 0x07+l]

	str := ""
	for _, ch := range name {
		if ch == 0x00 {
			break
		}
		if ch < 0x20 || ch >= 0x7f {
			return false, ""
		}

		if strings.Contains("$=?,[#:", string(ch)) {
			return false, ""
		}

		str += string(ch)
	}

	return true, str

}

type PascalVolumeHeader struct {
	data [PASCAL_DIRECTORY_ENTRY_LENGTH]byte
}

func (pvh *PascalVolumeHeader) SetData(data []byte) {
	for i, v := range data {
		if i < len(pvh.data) {
			pvh.data[i] = v
		}
	}
}

func (pvh *PascalVolumeHeader) GetStartBlock() int {
	return int(pvh.data[0x00]) + 256*int(pvh.data[0x01])
}

func (pvh *PascalVolumeHeader) GetNextBlock() int {
	return int(pvh.data[0x02]) + 256*int(pvh.data[0x03])
}

type PascalFileType int

const (
	FileType_PAS_NONE PascalFileType = 0
	FileType_PAS_BADD PascalFileType = 1
	FileType_PAS_CODE PascalFileType = 2
	FileType_PAS_TEXT PascalFileType = 3
	FileType_PAS_INFO PascalFileType = 4
	FileType_PAS_DATA PascalFileType = 5
	FileType_PAS_GRAF PascalFileType = 6
	FileType_PAS_FOTO PascalFileType = 7
	FileType_PAS_SECD PascalFileType = 8
)

var PascalTypeMap = map[PascalFileType][2]string{
	0x00: [2]string{"UNK", "ASCII Text"},
	0x01: [2]string{"BAD", "Bad Block"},
	0x02: [2]string{"PCD", "Pascal Code"},
	0x03: [2]string{"PTX", "Pascal Text"},
	0x04: [2]string{"PIF", "Pascal Info"},
	0x05: [2]string{"PDA", "Pascal Data"},
	0x06: [2]string{"GRF", "Pascal Graphics"},
	0x07: [2]string{"FOT", "HiRes Graphics"},
	0x08: [2]string{"SEC", "Secure Directory"},
}

func (ft PascalFileType) String() string {

	info, ok := PascalTypeMap[ft]
	if ok {
		return info[1]
	}

	return "Unknown"

}

func (ft PascalFileType) Ext() string {

	info, ok := PascalTypeMap[ft]
	if ok {
		return info[0]
	}

	return "UNK"

}

func PascalFileTypeFromExt(ext string) PascalFileType {
	for ft, info := range PascalTypeMap {
		if strings.ToUpper(ext) == info[0] {
			return ft
		}
	}
	return 0x00
}

func (pvh *PascalVolumeHeader) GetType() int {
	return int(int(pvh.data[0x04]) + 256*int(pvh.data[0x05]))
}

func (pvh *PascalVolumeHeader) GetNameLength() int {
	return int(pvh.data[0x06]) & 0x07
}

func (pvh *PascalVolumeHeader) GetName() string {
	l := pvh.GetNameLength()
	return string(pvh.data[0x07 : 0x07+l])
}

func (pvh *PascalVolumeHeader) GetTotalBlocks() int {
	return int(pvh.data[0x0e]) + 256*int(pvh.data[0x0f])
}

func (pvh *PascalVolumeHeader) GetNumFiles() int {
	return int(pvh.data[0x10]) + 256*int(pvh.data[0x11])
}

type PascalFileEntry struct {
	data [PASCAL_DIRECTORY_ENTRY_LENGTH]byte
}

func (pfe *PascalFileEntry) SetData(data []byte) {
	for i, v := range data {
		if i < len(pfe.data) {
			pfe.data[i] = v
		}
	}
}

func (pvh *PascalFileEntry) IsLocked() bool {
	return true
}

func (pvh *PascalFileEntry) GetStartBlock() int {
	return int(pvh.data[0x00]) + 256*int(pvh.data[0x01])
}

func (pvh *PascalFileEntry) GetNextBlock() int {
	return int(pvh.data[0x02]) + 256*int(pvh.data[0x03])
}

func (pvh *PascalFileEntry) GetType() PascalFileType {
	return PascalFileType(int(pvh.data[0x04]) + 256*int(pvh.data[0x05]))
}

func (pvh *PascalFileEntry) GetNameLength() int {
	return int(pvh.data[0x06]) & 0x0f
}

func (pvh *PascalFileEntry) GetName() string {
	l := pvh.GetNameLength()
	return string(pvh.data[0x07 : 0x07+l])
}

func (pvh *PascalFileEntry) GetBytesRemaining() int {
	return int(pvh.data[0x16]) + 256*int(pvh.data[0x17])
}

func (pvh *PascalFileEntry) GetFileSize() int {
	return pvh.GetBytesRemaining() + (pvh.GetNextBlock()-pvh.GetStartBlock()-1)*PASCAL_BLOCK_SIZE
}

func (dsk *DSKWrapper) PascalGetCatalog(pattern string) ([]*PascalFileEntry, error) {

	pattern = strings.Replace(pattern, ".", "[.]", -1)
	pattern = strings.Replace(pattern, "*", ".*", -1)
	pattern = strings.Replace(pattern, "?", ".", -1)

	rx := regexp.MustCompile("(?i)" + pattern)

	files := make([]*PascalFileEntry, 0)

	//

	d, err := dsk.PRODOSGetBlock(PASCAL_VOLUME_BLOCK)
	if err != nil {
		return nil, err
	}

	pvh := &PascalVolumeHeader{}
	pvh.SetData(d)
	numBlocks := pvh.GetNextBlock() - PASCAL_VOLUME_BLOCK

	if numBlocks < 0 || numBlocks > PASCAL_OVERSIZE_DIR {
		return files, errors.New("Directory appears corrupt")
	}

	// disk catalog is okay
	catdata := make([]byte, 0)
	for block := PASCAL_VOLUME_BLOCK; block < PASCAL_VOLUME_BLOCK+numBlocks; block++ {
		data, err := dsk.PRODOSGetBlock(block)
		if err != nil {
			return files, err
		}
		catdata = append(catdata, data...)
	}

	dirPtr := PASCAL_DIRECTORY_ENTRY_LENGTH
	for i := 0; i < pvh.GetNumFiles(); i++ {
		b := catdata[dirPtr : dirPtr+PASCAL_DIRECTORY_ENTRY_LENGTH]
		fd := &PascalFileEntry{}
		fd.SetData(b)
		// add file

		if rx.MatchString(fd.GetName()) {

			files = append(files, fd)

		}

		// move
		dirPtr += PASCAL_DIRECTORY_ENTRY_LENGTH
	}

	return files, nil

}

func (dsk *DSKWrapper) PascalUsedBitmap() ([]bool, error) {

	activeBlocks := dsk.Format.BPD()

	used := make([]bool, activeBlocks)

	files, err := dsk.PascalGetCatalog("*")
	if err != nil {
		return used, err
	}

	for _, file := range files {

		length := file.GetNextBlock() - file.GetStartBlock()
		start := file.GetStartBlock()
		if start+length > activeBlocks {
			continue // file is bad
		}

		for block := start; block < start+length; block++ {
			used[block] = true
		}

	}

	return used, nil

}

func (dsk *DSKWrapper) PascalReadFile(file *PascalFileEntry) ([]byte, error) {

	activeSectors := dsk.Format.BPD()

	length := file.GetNextBlock() - file.GetStartBlock()
	start := file.GetStartBlock()

	// If file is damaged return nothing
	if start+length > activeSectors {
		return []byte(nil), nil
	}

	block := start
	data := make([]byte, 0)
	for block < start+length && len(data) < file.GetFileSize() {

		chunk, err := dsk.PRODOSGetBlock(block)
		if err != nil {
			return data, err
		}
		needed := file.GetFileSize() - len(data)
		if needed >= PASCAL_BLOCK_SIZE {
			data = append(data, chunk...)
		} else {
			data = append(data, chunk[:needed]...)
		}

		block++

	}

	return data, nil

}
