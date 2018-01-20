package disk

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type VDH struct {
	Data        []byte
	blockid     int
	blockoffset int
}

func (fd *VDH) CreateTime() time.Time {

	b := fd.Data[0x1C:0x20]

	return prodosStampBytesToTime(b)

}

func (fd *VDH) SetCreateTime(t time.Time) {

	b := timeToProdosStampBytes(t)
	for i, v := range b {
		fd.Data[0x1C+i] = v
	}

}

func (fd *VDH) SetData(data []byte, blockid, blockoffset int) {

	fd.blockid = blockid
	fd.blockoffset = blockoffset

	if fd.Data == nil && len(data) == 39 {
		fd.Data = data
		//Println("VDH: ")
		//Dump(data)
	}

	for i, v := range data {
		fd.Data[i] = v
	}
}

func (fd *VDH) SetName(name string) {

	name = strings.ToUpper(name)

	if len(name) > 15 {
		name = name[:15]
	}

	l := len(name)

	for i, v := range []byte(name) {
		fd.Data[1+i] = v
	}

	fd.Data[0] = (fd.Data[0] & 0xf0) | byte(l)

}

func (fd *VDH) SetStorageType(t ProDOSStorageType) {

	fd.Data[0] = (fd.Data[0] & 0x0f) | (byte(t) << 4)

}

func (fd *VDH) GetNameLength() int {
	return int(fd.Data[0] & 0xf)
}

func (fd *VDH) GetStorageType() ProDOSStorageType {
	return ProDOSStorageType((fd.Data[0]) >> 4)
}

func (fd *VDH) GetDirName() string {
	return fd.GetVolumeName()
}

func (fd *VDH) GetVolumeName() string {

	l := fd.GetNameLength()

	b := fd.Data[1 : 1+l]

	s := ""
	for _, v := range b {
		s += string(rune(PokeToAscii(uint(v), false)))
	}

	return strings.Trim(s, " ")

}

func (fd *VDH) GetVersion() int {
	return int(fd.Data[28])
}

func (fd *VDH) SetVersion(b int) {
	fd.Data[28] = byte(b)
}

func (fd *VDH) GetMinVersion() int {
	return int(fd.Data[29])
}

func (fd *VDH) SetMinVersion(b int) {
	fd.Data[29] = byte(b)
}

func (fd *VDH) GetAccess() ProDOSAccessMode {
	return ProDOSAccessMode(fd.Data[30])
}

func (fd *VDH) SetAccess(m ProDOSAccessMode) {
	fd.Data[30] = byte(m)
}

func (fd *VDH) GetEntryLength() int {
	return int(fd.Data[31])
}

func (fd *VDH) SetEntryLength(b int) {
	fd.Data[31] = byte(b & 0xff)
}

func (fd *VDH) GetEntriesPerBlock() int {
	return int(fd.Data[32])
}

func (fd *VDH) SetEntriesPerBlock(b int) {
	fd.Data[32] = byte(b & 0xff)
}

func (fd *VDH) GetFileCount() int {
	return int(fd.Data[33]) + 256*int(fd.Data[34])
}

func (fd *VDH) SetFileCount(c int) {
	fd.Data[33] = byte(c & 0xff)
	fd.Data[34] = byte(c / 0x100)
}

func (fd *VDH) GetBitmapPointer() int {
	return int(fd.Data[35]) + 256*int(fd.Data[36])
}

func (fd *VDH) GetTotalBlocks() int {
	return int(fd.Data[37]) + 256*int(fd.Data[38])
}

func (fd *VDH) SetTotalBlocks(b int) {
	fd.Data[37] = byte(b % 256)
	fd.Data[38] = byte(b / 256)
}

func (fd *VDH) GetDirParentPointer() int {
	return int(fd.Data[35]) + 256*int(fd.Data[35])
}

func (fd *VDH) SetDirParentPointer(b int) {
	fd.Data[35] = byte(b & 0xff)
	fd.Data[36] = byte(b / 0x100)
}

func (fd *VDH) GetDirParentEntry() int {
	return int(fd.Data[37])
}

func (fd *VDH) SetDirParentEntry(b int) {
	fd.Data[37] = byte(b & 0xff)
}

func (fd *VDH) GetDirParentEntryLength() int {
	return int(fd.Data[38])
}

func (fd *VDH) SetDirParentEntryLength(b int) {
	fd.Data[38] = byte(b & 0xff)
}

func (fd *VDH) Publish(dsk *DSKWrapper) error {
	bd, err := dsk.PRODOSGetBlock(fd.blockid)
	if err != nil {
		return err
	}
	for i, v := range fd.Data {
		bd[fd.blockoffset+i] = v
	}
	fmt.Printf("Writing dir header at block %d\n", fd.blockid)

	fmt.Printf("Data=%v\n", bd)

	return dsk.PRODOSWrite(fd.blockid, bd)
}

func (dsk *DSKWrapper) IsProDOS() (bool, DiskFormat, SectorOrder) {

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
			vdh, err := dsk.PRODOSGetVDH(2)
			if err != nil {
				return false, oldFormat, oldLayout
			}

			if vdh.GetTotalBlocks() == 280 && vdh.GetStorageType() == 0xf {
				return true, GetDiskFormat(DF_PRODOS), l
			}

		}

	} else if len(dsk.Data) == PRODOS_800KB_DISK_BYTES {

		layouts := []SectorOrder{SectorOrderDOS33, SectorOrderDOS33Alt, SectorOrderProDOS}

		for _, l := range layouts {

			dsk.Layout = l
			vdh, err := dsk.PRODOSGetVDH(2)
			if err != nil {
				return false, oldFormat, oldLayout
			}

			if vdh.GetTotalBlocks() == 1600 && vdh.GetStorageType() == 0xf {
				return true, GetDiskFormat(DF_PRODOS_800KB), l
			}

		}

	}

	return false, oldFormat, oldLayout

}

func (dsk *DSKWrapper) PRODOS800GetVolumeBitmap() (ProDOSVolumeBitmap, error) {

	var vb ProDOSVolumeBitmap

	vdh, err := dsk.PRODOS800GetVDH(2)
	if err != nil {
		return vb, err
	}

	b := vdh.GetBitmapPointer()

	data, err := dsk.PRODOS800GetBlock(b)
	if err != nil {
		return vb, err
	}

	//copy(vb[:], data)

	vb = ProDOSVolumeBitmap{
		Data:        data,
		blockid:     b,
		blockoffset: 0,
	}

	return vb, nil

}

func (dsk *DSKWrapper) PRODOSGetVolumeBitmap() (ProDOSVolumeBitmap, error) {

	var vb ProDOSVolumeBitmap

	vdh, err := dsk.PRODOSGetVDH(2)
	if err != nil {
		return vb, err
	}

	b := vdh.GetBitmapPointer()

	data, err := dsk.PRODOSGetBlock(b)
	if err != nil {
		return vb, err
	}

	//copy(vb[:], data)
	vb = ProDOSVolumeBitmap{
		Data:        data,
		blockid:     b,
		blockoffset: 0,
	}

	return vb, nil

}

type ProDOSVolumeBitmap struct {
	Data        []byte
	blockid     int
	blockoffset int
}

func (vb ProDOSVolumeBitmap) IsBlockFree(b int) bool {

	bidx := b / 8
	bit := 7 - (b % 8)
	mask := byte(1 << uint(bit))

	return (vb.Data[bidx] & mask) == mask

}

func (vb ProDOSVolumeBitmap) SetBlockFree(b int, free bool) {
	bidx := b / 8
	bit := 7 - (b % 8)
	setmask := byte(1 << uint(bit))
	clrmask := 0xff ^ setmask

	if free {
		vb.Data[bidx] = vb.Data[bidx] | setmask
	} else {
		vb.Data[bidx] = vb.Data[bidx] & clrmask
	}
}

// -- Prodos file descriptor
type ProDOSFileDescriptor struct {
	Data        []byte
	blockid     int
	blockoffset int
	entryNum    int
}

func (fd *ProDOSFileDescriptor) SetData(data []byte, blockid int, blockoffset int) {

	fd.blockid = blockid
	fd.blockoffset = blockoffset

	if fd.Data == nil && len(data) == 39 {
		fd.Data = data
		return
	}

	for i, v := range data {
		fd.Data[i] = v
	}
}

func (fd *ProDOSFileDescriptor) GetNameLength() int {
	return int(fd.Data[0] & 0xf)
}

func (fd *ProDOSFileDescriptor) GetStorageType() ProDOSStorageType {
	return ProDOSStorageType((fd.Data[0]) >> 4)
}

func (fd *ProDOSFileDescriptor) Name() string {

	l := fd.GetNameLength()

	b := fd.Data[1 : 1+l]

	s := ""
	for _, v := range b {
		s += string(rune(PokeToAscii(uint(v), false)))
	}

	s = strings.ToLower(strings.Trim(s, " "))

	switch fd.Type() {
	case FileType_PD_APP:
		s += ".a"
	case FileType_PD_INT:
		s += ".i"
	case FileType_PD_BIN:
		s += ".s"
	case FileType_PD_TXT:
		s += ".t"
	case FileType_PD_SYS:
		s += ".s"
	}

	return s

}

func (fd *ProDOSFileDescriptor) NameUnadorned() string {

	l := fd.GetNameLength()

	b := fd.Data[1 : 1+l]

	s := ""
	for _, v := range b {
		s += string(rune(PokeToAscii(uint(v), false)))
	}

	s = strings.ToLower(strings.Trim(s, " "))

	return s

}

func (fd *ProDOSFileDescriptor) SetName(name string) {

	name = strings.ToUpper(name)

	if len(name) > 15 {
		name = name[:15]
	}

	l := len(name)

	for i, v := range []byte(name) {
		fd.Data[1+i] = v
	}

	fd.Data[0] = (fd.Data[0] & 0xf0) | byte(l)

}

func (fd *ProDOSFileDescriptor) SetStorageType(t ProDOSStorageType) {

	fd.Data[0] = (fd.Data[0] & 0x0f) | (byte(t) << 4)

}

func (fd *ProDOSFileDescriptor) SetAccessMode(t ProDOSAccessMode) {

	fd.Data[0x1e] = byte(t)

}

func (fd *ProDOSFileDescriptor) AccessMode() ProDOSAccessMode {
	return ProDOSAccessMode(fd.Data[0x1e])
}

func (fd *ProDOSFileDescriptor) SetLocked(b bool) {
	accessMode := fd.AccessMode()
	if b {
		accessMode = accessMode & (AccessType_Changed | AccessType_Readable)
	} else {
		accessMode = accessMode | AccessType_Destroy | AccessType_Writable | AccessType_Rename
	}
	fd.SetAccessMode(accessMode)
}

func (fd *ProDOSFileDescriptor) IsLocked() bool {
	a := fd.AccessMode()
	return a&(AccessType_Destroy|AccessType_Rename|AccessType_Writable) == 0
}

func prodosStampBytesToTime(in []byte) time.Time {
	dbits := (int(in[0x01]) << 8) | int(in[0x00])
	day := dbits & 31
	month := (dbits >> 5) & 15
	year := (dbits >> 9) & 127
	tbits := (int(in[0x03]) << 8) | int(in[0x02])
	mins := tbits & 63
	hours := (tbits >> 8)

	if year < 70 {
		year += 100
	}
	year += 1900

	return time.Date(year, time.Month(month), day, hours, mins, 0, 0, time.Local)
}

func timeToProdosStampBytes(t time.Time) []byte {
	year, month, day, hour, minute := t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute()
	year = year - 1900
	if year > 99 {
		year -= 100
	}

	dbits := (year << 9) | (month << 5) | day
	tbits := (hour << 8) | minute

	return []byte{
		byte(dbits & 0xff),
		byte(dbits >> 8),
		byte(tbits & 0xff),
		byte(tbits >> 8),
	}

}

func (fd *ProDOSFileDescriptor) CreateTime() time.Time {

	b := fd.Data[0x18:0x1C]

	return prodosStampBytesToTime(b)

}

func (fd *ProDOSFileDescriptor) SetCreateTime(t time.Time) {

	b := timeToProdosStampBytes(t)
	for i, v := range b {
		fd.Data[0x18+i] = v
	}

}

func (fd *ProDOSFileDescriptor) ModTime() time.Time {

	b := fd.Data[0x21:0x25]

	return prodosStampBytesToTime(b)

}

func (fd *ProDOSFileDescriptor) SetModTime(t time.Time) {

	b := timeToProdosStampBytes(t)
	for i, v := range b {
		fd.Data[0x21+i] = v
	}

}

func (fd *ProDOSFileDescriptor) GetVersion() int {
	return int(fd.Data[0x1c])
}

func (fd *ProDOSFileDescriptor) SetVersion(b int) {
	fd.Data[0x1c] = byte(b)
}

func (fd *ProDOSFileDescriptor) GetMinVersion() int {
	return int(fd.Data[0x1d])
}

func (fd *ProDOSFileDescriptor) SetMinVersion(b int) {
	fd.Data[0x1d] = byte(b)
}

type ProDOSAccessMode byte

const (
	AccessType_Destroy  ProDOSAccessMode = 0x80
	AccessType_Rename   ProDOSAccessMode = 0x40
	AccessType_Changed  ProDOSAccessMode = 0x20
	AccessType_Writable ProDOSAccessMode = 0x02
	AccessType_Readable ProDOSAccessMode = 0x01
	//
	AccessType_Default ProDOSAccessMode = AccessType_Readable | AccessType_Writable | AccessType_Rename | AccessType_Destroy
)

type ProDOSStorageType byte

const (
	StorageType_Inactive      ProDOSStorageType = 0x0
	StorageType_Seedling      ProDOSStorageType = 0x1
	StorageType_Sapling       ProDOSStorageType = 0x2
	StorageType_Tree          ProDOSStorageType = 0x3
	StorageType_SubDir_File   ProDOSStorageType = 0xd
	StorageType_SubDir_Header ProDOSStorageType = 0xe
	StorageType_Volume_Header ProDOSStorageType = 0xf
)

type ProDOSFileType byte

const (
	FileType_PD_None      ProDOSFileType = 0x00
	FileType_PD_TXT       ProDOSFileType = 0x04
	FileType_PD_Directory ProDOSFileType = 0x0f
	FileType_PD_BIN       ProDOSFileType = 0x06
	FileType_PD_INT       ProDOSFileType = 0xfa
	FileType_PD_INT_Var   ProDOSFileType = 0xfb
	FileType_PD_APP       ProDOSFileType = 0xfc
	FileType_PD_APP_Var   ProDOSFileType = 0xfd
	FileType_PD_Reloc     ProDOSFileType = 0xfe
	FileType_PD_SYS       ProDOSFileType = 0xff
)

var ProDOSTypeMap = map[ProDOSFileType][2]string{
	0x00: [2]string{"UNK", "Unknown"},
	0x01: [2]string{"BAD", "Bad Block"},
	0x02: [2]string{"PCD", "Pascal Code"},
	0x03: [2]string{"PTX", "Pascal Text"},
	0x04: [2]string{"TXT", "ASCII Text"},
	0x05: [2]string{"PDA", "Pascal Data"},
	0x06: [2]string{"BIN", "Binary File"},
	0x07: [2]string{"FNT", "Apple III Font"},
	0x08: [2]string{"FOT", "HiRes/Double HiRes Graphics"},
	0x09: [2]string{"BA3", "Apple III Basic Program"},
	0x0A: [2]string{"DA3", "Apple III Basic Data"},
	0x0B: [2]string{"WPF", "Generic Word Processing"},
	0x0C: [2]string{"SOS", "SOS System File"},
	0x0F: [2]string{"DIR", "ProDOS Directory"},
	0x10: [2]string{"RPD", "RPS Data"},
	0x11: [2]string{"RPI", "RPS Index"},
	0x12: [2]string{"AFD", "AppleFile Discard"},
	0x13: [2]string{"AFM", "AppleFile Model"},
	0x14: [2]string{"AFR", "AppleFile Report"},
	0x15: [2]string{"SCL", "Screen Library"},
	0x16: [2]string{"PFS", "PFS Document"},
	0x19: [2]string{"ADB", "AppleWorks Database"},
	0x1A: [2]string{"AWP", "AppleWorks Word Processing"},
	0x1B: [2]string{"ASP", "AppleWorks Spreadsheet"},
	0x80: [2]string{"GES", "System File"},
	0x81: [2]string{"GEA", "Desk Accessory"},
	0x82: [2]string{"GEO", "Application"},
	0x83: [2]string{"GED", "Document"},
	0x84: [2]string{"GEF", "Font"},
	0x85: [2]string{"GEP", "Printer Driver"},
	0x86: [2]string{"GEI", "Input Driver"},
	0x87: [2]string{"GEX", "Auxiliary Driver"},
	0x89: [2]string{"GEV", "Swap File"},
	0x8B: [2]string{"GEC", "Clock Driver"},
	0x8C: [2]string{"GEK", "Interface Card Driver"},
	0x8D: [2]string{"GEW", "Formatting Data"},
	0xA0: [2]string{"WP", " WordPerfect"},
	0xAB: [2]string{"GSB", "Apple IIgs BASIC Program"},
	0xAC: [2]string{"TDF", "Apple IIgs BASIC TDF"},
	0xAD: [2]string{"BDF", "Apple IIgs BASIC Data"},
	0x60: [2]string{"PRE", "PC Pre-Boot"},
	0x6B: [2]string{"BIO", "PC BIOS"},
	0x66: [2]string{"NCF", "ProDOS File Navigator Command File"},
	0x6D: [2]string{"DVR", "PC Driver"},
	0x6E: [2]string{"PRE", "PC Pre-Boot"},
	0x6F: [2]string{"HDV", "PC Hard Disk Image"},
	0x50: [2]string{"GWP", "Apple IIgs Word Processing"},
	0x51: [2]string{"GSS", "Apple IIgs Spreadsheet"},
	0x52: [2]string{"GDB", "Apple IIgs Database"},
	0x53: [2]string{"DRW", "Object Oriented Graphics"},
	0x54: [2]string{"GDP", "Apple IIgs Desktop Publishing"},
	0x55: [2]string{"HMD", "HyperMedia"},
	0x56: [2]string{"EDU", "Educational Program Data"},
	0x57: [2]string{"STN", "Stationery"},
	0x58: [2]string{"HLP", "Help File"},
	0x59: [2]string{"COM", "Communications"},
	0x5A: [2]string{"CFG", "Configuration"},
	0x5B: [2]string{"ANM", "Animation"},
	0x5C: [2]string{"MUM", "Multimedia"},
	0x5D: [2]string{"ENT", "Entertainment"},
	0x5E: [2]string{"DVU", "Development Utility"},
	0x41: [2]string{"OCR", "Optical Character Recognition"},
	0x42: [2]string{"FTD", "File Type Definitions"},
	0x20: [2]string{"TDM", "Desktop Manager File"},
	0x21: [2]string{"IPS", "Instant Pascal Source"},
	0x22: [2]string{"UPV", "UCSD Pascal Volume"},
	0x29: [2]string{"3SD", "SOS Directory"},
	0x2A: [2]string{"8SC", "Source Code"},
	0x2B: [2]string{"8OB", "Object Code"},
	0x2C: [2]string{"8IC", "Interpreted Code"},
	0x2D: [2]string{"8LD", "Language Data"},
	0x2E: [2]string{"P8C", "ProDOS 8 Code Module"},
	0xB0: [2]string{"SRC", "Apple IIgs Source Code"},
	0xB1: [2]string{"OBJ", "Apple IIgs Object Code"},
	0xB2: [2]string{"LIB", "Apple IIgs Library"},
	0xB3: [2]string{"S16", "Apple IIgs Application Program"},
	0xB4: [2]string{"RTL", "Apple IIgs Runtime Library"},
	0xB5: [2]string{"EXE", "Apple IIgs Shell Script"},
	0xB6: [2]string{"PIF", "Apple IIgs Permanent INIT"},
	0xB7: [2]string{"TIF", "Apple IIgs Temporary INIT"},
	0xB8: [2]string{"NDA", "Apple IIgs New Desk Accessory"},
	0xB9: [2]string{"CDA", "Apple IIgs Classic Desk Accessory"},
	0xBA: [2]string{"TOL", "Apple IIgs Tool"},
	0xBB: [2]string{"DRV", "Apple IIgs Device Driver"},
	0xBC: [2]string{"LDF", "Apple IIgs Generic Load File"},
	0xBD: [2]string{"FST", "Apple IIgs File System Translator"},
	0xBF: [2]string{"DOC", "Apple IIgs Document   "},
	0xC0: [2]string{"PNT", "Apple IIgs Packed Super HiRes"},
	0xC1: [2]string{"PIC", "Apple IIgs Super HiRes"},
	0xC2: [2]string{"ANI", "PaintWorks Animation"},
	0xC3: [2]string{"PAL", "PaintWorks Palette"},
	0xC5: [2]string{"OOG", "Object-Oriented Graphics"},
	0xC6: [2]string{"SCR", "Script"},
	0xC7: [2]string{"CDV", "Apple IIgs Control Panel"},
	0xC8: [2]string{"FON", "Apple IIgs Font"},
	0xC9: [2]string{"FND", "Apple IIgs Finder Data"},
	0xCA: [2]string{"ICN", "Apple IIgs Icon "},
	0xD5: [2]string{"MUS", "Music"},
	0xD6: [2]string{"INS", "Instrument"},
	0xD7: [2]string{"MDI", "MIDI"},
	0xD8: [2]string{"SND", "Apple IIgs Audio"},
	0xDB: [2]string{"DBM", "DB Master Document"},
	0xE0: [2]string{"LBR", "Archive"},
	0xE2: [2]string{"ATK", "AppleTalk Data"},
	0xEE: [2]string{"R16", "EDASM 816 Relocatable Code"},
	0xEF: [2]string{"PAR", "Pascal Area"},
	0xF0: [2]string{"CMD", "ProDOS Command File"},
	0xF1: [2]string{"OVL", "User Defined 1"},
	0xF2: [2]string{"UD2", "User Defined 2"},
	0xF3: [2]string{"UD3", "User Defined 3"},
	0xF4: [2]string{"UD4", "User Defined 4"},
	0xF5: [2]string{"BAT", "User Defined 5"},
	0xF6: [2]string{"UD6", "User Defined 6"},
	0xF7: [2]string{"UD7", "User Defined 7"},
	0xF8: [2]string{"PRG", "User Defined 8"},
	0xF9: [2]string{"P16", "ProDOS-16 System File"},
	0xFA: [2]string{"INT", "Integer BASIC Program"},
	0xFB: [2]string{"IVR", "Integer BASIC Variables"},
	0xFC: [2]string{"BAS", "Applesoft BASIC Program"},
	0xFD: [2]string{"VAR", "Applesoft BASIC Variables"},
	0xFE: [2]string{"REL", "EDASM Relocatable Code"},
	0xFF: [2]string{"SYS", "ProDOS-8 System File"},
}

func (t ProDOSFileType) String() string {

	info, ok := ProDOSTypeMap[t]
	if ok {
		return info[1]
	}

	return "Unknown"
}

func (ft ProDOSFileType) Ext() string {

	info, ok := ProDOSTypeMap[ft]
	if ok {
		return info[0]
	}

	return "BIN"
}

func ProDOSFileTypeFromExt(ext string) ProDOSFileType {
	for ft, info := range ProDOSTypeMap {
		if strings.ToUpper(ext) == info[0] {
			return ft
		}
	}
	return 0x06
}

func (t ProDOSFileType) Valid() bool {

	if t == FileType_PD_None {
		return false
	}

	_, ok := ProDOSTypeMap[t]

	return ok
}

func (fd *ProDOSFileDescriptor) Type() ProDOSFileType {
	return ProDOSFileType(fd.Data[16])
}

func (fd *ProDOSFileDescriptor) SetType(t ProDOSFileType) {
	fd.Data[16] = byte(t)
}

func (fd *ProDOSFileDescriptor) IndexBlock() int {
	return int(fd.Data[17]) + 256*int(fd.Data[18])
}

func (fd *ProDOSFileDescriptor) SetIndexBlock(b int) {
	fd.Data[17] = byte(b & 0xff)
	fd.Data[18] = byte(b / 256)
}

func (fd *ProDOSFileDescriptor) TotalBlocks() int {
	return int(fd.Data[19]) + 256*int(fd.Data[20])
}

func (fd *ProDOSFileDescriptor) SetTotalBlocks(b int) {
	fd.Data[19] = byte(b & 0xff)
	fd.Data[20] = byte(b / 256)
}

func (fd *ProDOSFileDescriptor) AuxType() int {
	return int(fd.Data[31]) + 256*int(fd.Data[32])
}

func (fd *ProDOSFileDescriptor) SetAuxType(b int) {
	fd.Data[31] = byte(b & 0xff)
	fd.Data[32] = byte(b / 256)
}

func (fd *ProDOSFileDescriptor) TotalSectors() int {
	return fd.TotalBlocks() / 2
}

func (fd *ProDOSFileDescriptor) Size() int {
	return int(fd.Data[21]) + 256*int(fd.Data[22]) + 65536*int(fd.Data[23])
}

func (fd *ProDOSFileDescriptor) SetSize(v int) {
	fd.Data[21] = byte(v & 0xff)
	fd.Data[22] = byte((v >> 8) & 0xff)
	fd.Data[23] = byte((v >> 16) & 0xff)
}

func (fd *ProDOSFileDescriptor) SetHeaderPointer(v int) {
	fd.Data[0x25] = byte(v & 0xff)
	fd.Data[0x26] = byte((v >> 8) & 0xff)
}

func (fd *ProDOSFileDescriptor) HeaderPointer() int {
	return int(fd.Data[0x25]) + 256*int(fd.Data[0x26])
}

func (d *DSKWrapper) PRODOS800GetBlock(block int) ([]byte, error) {

	t, s1, s2 := d.PRODOS800GetBlockSectors(block)

	e := d.Seek(t, s1)
	if e != nil {
		return []byte(nil), e
	}
	data := make([]byte, 0)
	c1 := d.Read()
	data = append(data, c1...)

	e = d.Seek(t, s2)
	if e != nil {
		return []byte(nil), e
	}
	c2 := d.Read()
	data = append(data, c2...)

	return data, nil
}

func (d *DSKWrapper) PRODOSGetBlock(block int) ([]byte, error) {

	t, s1, s2 := d.PRODOSGetBlockSectors(block)

	e := d.Seek(t, s1)
	if e != nil {
		return []byte(nil), e
	}
	data := make([]byte, 0)
	c1 := d.Read()
	data = append(data, c1...)

	e = d.Seek(t, s2)
	if e != nil {
		return []byte(nil), e
	}
	c2 := d.Read()
	data = append(data, c2...)

	return data, nil
}

func (d *DSKWrapper) PRODOS800GetVDH(b int) (*VDH, error) {

	data, e := d.PRODOS800GetBlock(b)
	if e != nil {
		return nil, e
	}

	vdh := &VDH{}
	vdh.SetData(data[4:43], b, 4)
	return vdh, nil

}

func (d *DSKWrapper) PRODOSGetVDH(b int) (*VDH, error) {

	data, e := d.PRODOSGetBlock(b)
	if e != nil {
		return nil, e
	}

	vdh := &VDH{}
	vdh.SetData(data[4:43], b, 4)
	return vdh, nil

}

func (d *DSKWrapper) PRODOSGetCatalogPathed(start int, path string, pattern string) (*VDH, []ProDOSFileDescriptor, error) {

	path = strings.Trim(path, "/")
	//fmt.Printf("PRODOSGetCatalogPathed(%d, \"%s\", \"%s\" )\n", start, path, pattern)

	//start := 2 // where we start our descent

	if path != "" {
		parts := strings.Split(strings.Trim(path, "/"), "/")

		subdir := parts[0]
		parts = parts[1:]

		// find subdirectories
		vdh, files, e := d.PRODOSGetCatalog(start, subdir)
		if e != nil {
			return vdh, files, e
		}
		if len(files) != 1 {
			return vdh, files, e
		}
		if files[0].Type() != FileType_PD_Directory {
			return vdh, files, errors.New("Not a directory")
		}

		// ok, we found the directory...
		if len(parts) > 0 {
			// more subdirs
			//fmt.Printf(">> Entering prodos subdir [%s]\n", files[0].Name())
			newpathstr := strings.Join(parts, "/")
			return d.PRODOSGetCatalogPathed(files[0].IndexBlock(), newpathstr, pattern)
		} else {
			// get directory based on this block
			//fmt.Printf("-- Entering prodos subdir [%s]\n", files[0].Name())
			return d.PRODOSGetCatalog(files[0].IndexBlock(), pattern)
		}

	} else {
		return d.PRODOSGetCatalog(start, pattern)
	}

}

func (d *DSKWrapper) PRODOSGetCatalog(startblock int, pattern string) (*VDH, []ProDOSFileDescriptor, error) {

	//fmt.Printf("GetCatalogProDOS(%d, %s)\n", startblock, pattern)

	var err error
	var re *regexp.Regexp
	var patterntmp string
	if pattern != "" {
		patterntmp = strings.Replace(pattern, ".", "[.]", -1)
		patterntmp = strings.Replace(patterntmp, "*", ".*", -1)
		patterntmp = "(?i)^" + patterntmp + "$"
		re = regexp.MustCompile(patterntmp)
	}

	var files []ProDOSFileDescriptor
	var e error
	var vtoc *VDH

	if d.Format.ID == DF_PRODOS_800KB {
		vtoc, e = d.PRODOS800GetVDH(startblock)
	} else {
		vtoc, e = d.PRODOSGetVDH(startblock)
	}
	if e != nil {
		return vtoc, files, e
	}

	activeentries := 0
	filecount := vtoc.GetFileCount()
	blockentries := 2
	entriesperblock := vtoc.GetEntriesPerBlock()
	refnum := startblock

	var data []byte
	if d.Format.ID == DF_PRODOS_800KB {
		data, _ = d.PRODOS800GetBlock(refnum)
	} else {
		data, _ = d.PRODOSGetBlock(refnum)
	}

	nextblock := int(data[2]) + 256*int(data[3])

	//fmt.Printf("ActiveCount = %d\n", filecount)

	entrypointer := 4 + PRODOS_ENTRY_SIZE

	for activeentries < filecount {

		if data[entrypointer] != 0x00 {
			// Valid entry
			chunk := data[entrypointer : entrypointer+PRODOS_ENTRY_SIZE]
			fd := ProDOSFileDescriptor{}
			fd.SetData(chunk, refnum, entrypointer)

			if fd.Type().Valid() {

				var skipname bool = false
				if re != nil {
					//fmt.Printf("Checking [%s] against regex /%s/\n", fd.Name(), patterntmp)
					skipname = !re.MatchString(fd.Name())
				}

				if fd.GetStorageType() != StorageType_Inactive && !skipname {
					files = append(files, fd)
				}

			}

			activeentries++
		}

		if activeentries < filecount {
			if blockentries == entriesperblock {
				refnum = nextblock
				if d.Format.ID == DF_PRODOS_800KB {
					data, err = d.PRODOS800GetBlock(refnum)
				} else {
					data, err = d.PRODOSGetBlock(refnum)
				}
				if err != nil {
					break
				}
				nextblock = int(data[2]) + 256*int(data[3])
				blockentries = 0x01
				entrypointer = 0x04
			} else {
				entrypointer += PRODOS_ENTRY_SIZE
				blockentries++
			}
		}

	}

	return vtoc, files, nil
}

func (d *DSKWrapper) PRODOSReadFileSectors(fd ProDOSFileDescriptor, maxblocks int) ([]byte, error) {

	var data, index, chunk []byte
	var e error

	switch fd.GetStorageType() {
	case StorageType_Seedling:
		/* single block pointed to */
		if d.Format.ID == DF_PRODOS_800KB {
			data, _ = d.PRODOS800GetBlock(fd.IndexBlock())
		} else {
			data, _ = d.PRODOSGetBlock(fd.IndexBlock())
		}
		count := fd.Size()
		if count > len(data) {
			count = len(data)
		}
		return data[:count], e
	case StorageType_Sapling:
		if d.Format.ID == DF_PRODOS_800KB {
			index, _ = d.PRODOS800GetBlock(fd.IndexBlock())
		} else {
			index, _ = d.PRODOSGetBlock(fd.IndexBlock())
		}
		data := make([]byte, 0)
		bptr := 0
		remaining := fd.Size() - len(data)
		for len(data) < fd.Size() && bptr+256 < len(index) {
			blocknum := int(index[bptr]) + 256*int(index[bptr+256])

			//fmt.Printf("File block %d (%d %d)\n", blocknum, int(index[bptr]), int(index[bptr+1]))

			if d.Format.ID == DF_PRODOS_800KB {
				chunk, e = d.PRODOS800GetBlock(blocknum)
			} else {
				chunk, e = d.PRODOSGetBlock(blocknum)
			}
			if e != nil {
				return data, e
			}

			count := 512
			if remaining < count {
				count = remaining
			}

			data = append(data, chunk[:count]...)

			bptr += 1
			remaining = fd.Size() - len(data)
		}
		return data, e
	case StorageType_Tree:
		return []byte(nil), errors.New("Unimplemented yet... soz :(")
	}

	return []byte(nil), nil
}

func (d *DSKWrapper) PRODOSReadFile(fd ProDOSFileDescriptor) (int, int, []byte, error) {

	data, e := d.PRODOSReadFileSectors(fd, -1)

	if e != nil {
		return 0, 0, data, e
	}

	switch fd.Type() {
	case FileType_PD_INT:
		return fd.Size(), fd.AuxType(), IntegerDetoks(data), nil
	case FileType_PD_APP:
		return fd.Size(), fd.AuxType(), ApplesoftDetoks(data), nil
	case FileType_PD_TXT:
		return fd.Size(), fd.AuxType(), data, nil
	case FileType_PD_BIN:
		return fd.Size(), fd.AuxType(), data, nil
	default:
		return fd.Size(), fd.AuxType(), data, nil
	}

}

func (d *DSKWrapper) PRODOSReadFileRaw(fd ProDOSFileDescriptor) (int, int, []byte, error) {

	data, e := d.PRODOSReadFileSectors(fd, -1)

	if e != nil {
		return 0, 0, data, e
	}

	switch fd.Type() {
	case FileType_PD_INT:
		return fd.Size(), fd.AuxType(), data, nil
	case FileType_PD_APP:
		return fd.Size(), fd.AuxType(), data, nil
	case FileType_PD_TXT:
		return fd.Size(), fd.AuxType(), data, nil
	case FileType_PD_BIN:
		return fd.Size(), fd.AuxType(), data, nil
	default:
		return fd.Size(), fd.AuxType(), data, nil
	}

}

func (d *DSKWrapper) PRODOSChecksumBlock(b int) string {
	bl, _ := d.PRODOSGetBlock(b)
	return Checksum(bl)
}

func (d *DSKWrapper) PRODOS800ChecksumBlock(b int) string {
	bl, _ := d.PRODOS800GetBlock(b)
	return Checksum(bl)
}

func (d *DSKWrapper) PRODOSGetBlockSectors(block int) (int, int, int) {

	track := block / PRODOS_BLOCKS_PER_TRACK

	bo := block % PRODOS_BLOCKS_PER_TRACK

	if d.Layout == SectorOrderProDOSLinear {
		return track, bo * 2, bo*2 + 1
	}

	switch bo {
	case 0:
		return track, 0x0, 0xe
	case 1:
		return track, 0xd, 0xc
	case 2:
		return track, 0xb, 0xa
	case 3:
		return track, 0x9, 0x8
	case 4:
		return track, 0x7, 0x6
	case 5:
		return track, 0x5, 0x4
	case 6:
		return track, 0x3, 0x2
	case 7:
		return track, 0x1, 0xf
	}

	return track, 0x0, 0xe

}

func (d *DSKWrapper) PRODOS800GetBlockSectors(block int) (int, int, int) {

	dblBlock := block * 2

	spt := 40

	track := dblBlock / spt
	s1 := dblBlock % spt
	s2 := (dblBlock + 1) % spt

	return track, s1, s2

}

// PRODOSGetDirBlocks returns a list of blocks for the current directory level
func (dsk *DSKWrapper) PRODOSGetDirBlocks(start int) (*VDH, []int, [][]byte, error) {
	blocks := make([]int, 0)
	chunks := make([][]byte, 0)

	vdh, err := dsk.PRODOSGetVDH(start)
	if err != nil {
		return vdh, blocks, chunks, err
	}

	// okay lets trail the directory
	blocks = append(blocks, start)
	data, _ := dsk.PRODOSGetBlock(start)
	chunks = append(chunks, data)
	nextBlock := int(data[0x02]) + 256*int(data[0x03])
	for nextBlock != 0 {
		blocks = append(blocks, nextBlock)
		data, _ := dsk.PRODOSGetBlock(nextBlock)
		chunks = append(chunks, data)
		nextBlock = int(data[0x02]) + 256*int(data[0x03])
	}

	return vdh, blocks, chunks, nil

}

// PRODOSFindDirBlocks locates the blocks for a given directory (if it exists!)
func (dsk *DSKWrapper) PRODOSFindDirBlocks(start int, path string) (*VDH, []int, [][]byte, error) {

	path = strings.Trim(path, "/")

	if path == "" {
		return dsk.PRODOSGetDirBlocks(start)
	}

	segments := strings.Split(path, "/")
	target := segments[0]
	segments = segments[1:]

	vdh, files, err := dsk.PRODOSGetCatalog(start, "")
	if err != nil {
		return vdh, nil, nil, err
	}

	for _, f := range files {

		if f.Type() != FileType_PD_Directory {
			continue
		}

		if strings.ToLower(target) == strings.ToLower(f.NameUnadorned()) {
			// matched path
			newpath := strings.Join(segments, "/")
			return dsk.PRODOSFindDirBlocks(f.IndexBlock(), newpath)
		}

	}

	return vdh, nil, nil, errors.New("Path not found")

}

// PRODOSGetFirstFreeEntry for a given path, will find the first free file descriptor
// it will expand the directory if needed to create a free block
func (dsk *DSKWrapper) PRODOSGetFirstFreeEntry(path string, name string, grow bool) (*ProDOSFileDescriptor, error) {

	vdh, blockList, blockData, err := dsk.PRODOSFindDirBlocks(2, path)
	if err != nil {
		return nil, err
	}

	entries := 0

	// if we are here, we found the right directory...
	for idx, data := range blockData {
		count := 0

		if idx == 0 {
			count = 1
			entries = 1
		}

		for count < vdh.GetEntriesPerBlock() {
			offset := 4 + count*PRODOS_ENTRY_SIZE
			chunk := data[offset : offset+PRODOS_ENTRY_SIZE]
			fd := &ProDOSFileDescriptor{}
			fd.SetData(chunk, blockList[idx], offset)
			fd.entryNum = entries

			//Printf("--> Check entry: %s, %v, %v\n", fd.NameUnadorned(), fd.CreateTime(), fd.ModTime())

			if fd.GetStorageType() != 0x00 && strings.ToLower(fd.NameUnadorned()) == strings.ToLower(name) {
				return fd, nil
			} else if fd.GetStorageType() == 0x00 {
				//fmt.Printf("found at entry %d in block %d\n", entries, blockList[idx])
				return fd, nil
			}

			count += 1
			entries++
		}
	}

	if !grow {
		return nil, errors.New("No free slot: told not to grow directory")
	}

	// If we got here, we need to create a new block
	freeBlocks, err := dsk.PRODOSGetFreeBlocks(1, vdh.GetTotalBlocks())
	if err != nil {
		return nil, errors.New("Could not extend directory")
	}

	data, err := dsk.PRODOSGetBlock(freeBlocks[0])
	prevBlock := blockList[len(blockList)-1]
	data[0x00] = byte(prevBlock & 0xff)
	data[0x01] = byte(prevBlock / 0x100)

	offset := 4
	chunk := data[offset : offset+PRODOS_ENTRY_SIZE]
	fd := &ProDOSFileDescriptor{}
	fd.entryNum = entries
	fd.SetData(chunk, freeBlocks[0], offset)

	dsk.PRODOSMarkBlocks(freeBlocks, false)

	return fd, nil
}

func (dsk *DSKWrapper) PRODOSMarkBlocks(list []int, free bool) error {
	vbm, err := dsk.PRODOSGetVolumeBitmap()
	if err != nil {
		return err
	}

	for _, b := range list {
		vbm.SetBlockFree(b, free)
	}

	//fmt.Printf("Writing Volume bitmap to block %d\n", vbm.blockid)

	return dsk.PRODOSWrite(vbm.blockid, vbm.Data)
}

func (dsk *DSKWrapper) PRODOSGetFreeBlocks(count int, totalBlocks int) ([]int, error) {

	vbm, err := dsk.PRODOSGetVolumeBitmap()
	if err != nil {
		return nil, err
	}
	b := 0

	blocks := make([]int, 0)

	for b < totalBlocks && len(blocks) < count {
		if vbm.IsBlockFree(b) {
			blocks = append(blocks, b)
		}
		b++
	}

	if len(blocks) == count {
		return blocks, nil
	}

	return blocks, errors.New("Not enough blocks")

}

func (dsk *DSKWrapper) PRODOSDeleteFile(path string, name string) error {

	fd, err := dsk.PRODOSGetNamedEntry(path, name)
	if err != nil {
		return err
	}

	if fd.GetStorageType() == 0x00 {
		return errors.New("Not found")
	}

	// At this stage we have a match
	access := fd.AccessMode()
	if access&AccessType_Destroy == 0 {
		return errors.New("Permission denied")
	}
	if access&AccessType_Writable == 0 {
		return errors.New("Read-only file")
	}

	// Make sure its either a sapling or a seedling
	st := fd.GetStorageType()
	if st != StorageType_Sapling && st != StorageType_Seedling && st != StorageType_SubDir_File {
		return errors.New("Special file deletion not implemented: yet.")
	} else if st == StorageType_SubDir_File {
		return dsk.PRODOSDeleteDirectory(path, name)
	}

	var removeBlocks []int
	switch st {
	case StorageType_Seedling:
		removeBlocks = append(removeBlocks, fd.IndexBlock())
	case StorageType_Sapling:
		removeBlocks = append(removeBlocks, fd.IndexBlock())
		ib, err := dsk.PRODOSGetBlock(removeBlocks[0])
		if err != nil {
			return err
		}

		i := 0
		b := int(ib[2*i+0]) + 256*int(ib[2*i+1])
		for i < 256 && b != 0 {
			b = int(ib[2*i+0]) + 256*int(ib[2*i+1])
			i++
		}
	}

	err = dsk.PRODOSMarkBlocks(removeBlocks, true)
	if err != nil {
		return err
	}

	// // get the VDH -- we need this later
	vdh, _, _, err := dsk.PRODOSFindDirBlocks(2, path)
	if err != nil {
		return err
	}

	// now delete the fileentry
	fd.SetStorageType(StorageType_Inactive)
	err = fd.Publish(dsk)
	if err != nil {
		return err
	}

	// update filecount
	vdh.SetFileCount(vdh.GetFileCount() - 1)

	return vdh.Publish(dsk)

}

func (dsk *DSKWrapper) PRODOSWriteFile(path string, name string, kind ProDOSFileType, data []byte, auxtype int) error {

	name = strings.ToUpper(name)

	nst := StorageType_Seedling
	blocksNeeded := len(data)/512 + 1
	totalBlocks := blocksNeeded
	if blocksNeeded > 1 {
		nst = StorageType_Sapling
		totalBlocks++ // extra block for blocklist
	} else if blocksNeeded > 256 {
		return errors.New("Not implemented: Tree write")
	}

	var origTime time.Time
	var origAccess ProDOSAccessMode

	fd, err := dsk.PRODOSGetNamedEntry(path, name)
	if err == nil {
		origTime = fd.CreateTime() // we need this later
		origAccess = fd.AccessMode()
		err = dsk.PRODOSDeleteFile(path, name)
		if err != nil {
			return err
		}
	} else {

		fd, err = dsk.PRODOSGetFirstFreeEntry(path, name, true)
		if err != nil {
			return err
		}

	}

	vdh, err := dsk.PRODOSGetVDH(2)
	if err != nil {
		return err
	}
	freeBlocks, err := dsk.PRODOSGetFreeBlocks(totalBlocks, vdh.GetTotalBlocks())
	if err != nil {
		return err
	}

	// Okay got enough blocks
	switch nst {
	case StorageType_Sapling:
		err = dsk.PRODOSWriteSaplingBlocks(freeBlocks[0], freeBlocks[1:], data)
		if err != nil {
			return err
		}
	case StorageType_Seedling:
		//fmt.Printf("Write Seedling %d bytes data to block %d\n", len(data), freeBlocks[0])
		err = dsk.PRODOSWrite(freeBlocks[0], data)
		if err != nil {
			return err
		}
	}

	// Get the current directories directory header
	dvdh, blocks, _, err := dsk.PRODOSFindDirBlocks(2, path)
	if err != nil {
		return err
	}

	// common details - note we preserve access mode and time when overwriting the same file
	fd.SetAuxType(auxtype)
	fd.SetName(name)
	fd.SetType(kind)
	fd.SetTotalBlocks(totalBlocks)
	fd.SetIndexBlock(freeBlocks[0])
	fd.SetSize(len(data))
	fd.SetStorageType(nst)
	if origAccess == 0x00 {
		fd.SetAccessMode(AccessType_Default)
	} else {
		fd.SetAccessMode(origAccess)
	}
	if origTime.IsZero() {
		fd.SetCreateTime(time.Now())
	} else {
		fd.SetCreateTime(origTime)
	}
	fd.SetModTime(time.Now())
	fd.SetHeaderPointer(blocks[0])

	fd.Publish(dsk)

	dvdh.SetFileCount(vdh.GetFileCount() + 1)
	dvdh.Publish(dsk)

	err = dsk.PRODOSMarkBlocks(freeBlocks, false)
	if err != nil {
		return err
	}

	return nil
}

func (fd *ProDOSFileDescriptor) Publish(dsk *DSKWrapper) error {

	//fmt.Printf("Writing FD data back to block %d, offset %d\n", fd.blockid, fd.blockoffset)

	bd, err := dsk.PRODOSGetBlock(fd.blockid)
	if err != nil {
		return err
	}
	for i, v := range fd.Data {
		bd[fd.blockoffset+i] = v
	}
	// for i, _ := range bd {
	// 	bd[i] = byte(0xff ^ (i % 2))
	// }

	//Dump(bd)

	return dsk.PRODOSWrite(fd.blockid, bd)
}

func (dsk *DSKWrapper) PRODOSWrite(b int, data []byte) error {

	for len(data) < 512 {
		data = append(data, 0x00)
	}

	t, s1, s2 := dsk.PRODOSGetBlockSectors(b)

	err := dsk.Seek(t, s1)
	if err != nil {
		return err
	}
	dsk.Write(data[:256])
	err = dsk.Seek(t, s2)
	if err != nil {
		return err
	}
	dsk.Write(data[256:])

	return nil

}

func (dsk *DSKWrapper) PRODOSWriteSaplingBlocks(indexBlock int, dataBlocks []int, data []byte) error {

	if len(dataBlocks) > 256 || len(data) > 0x20000 {
		return errors.New("Too many data blocks")
	}

	ib := make([]byte, 512)
	for i, blocknum := range dataBlocks {
		// index the block
		ib[i*2+0] = byte(blocknum & 0xff)
		ib[i*2+1] = byte(blocknum / 0x100)

		// data offset...
		ptr := 512 * i
		end := ptr + 512
		if end > len(data) {
			end = len(data)
		}
		chunk := data[ptr:end]
		for len(chunk) < 512 {
			chunk = append(chunk, 0x00)
		}
		err := dsk.PRODOSWrite(blocknum, chunk)
		if err != nil {
			return err
		}
	}

	return dsk.PRODOSWrite(indexBlock, ib)
}

// PRODOSCreateDirectory tries to create a subdirectory...
func (dsk *DSKWrapper) PRODOSCreateDirectory(path string, name string) error {

	vdh, err := dsk.PRODOSGetVDH(2)
	if err != nil {
		return err
	}

	fd, err := dsk.PRODOSGetNamedEntry(path, name)
	if err == nil {
		return errors.New("Item exists")
	}

	fd, err = dsk.PRODOSGetFirstFreeEntry(path, name, true)
	if err != nil {
		return err
	}

	// got file descriptor
	if fd.GetStorageType() != 0 {

		if fd.Type() == FileType_PD_Directory {
			return errors.New("Directory already exists")
		}

		return errors.New("Type mismatch")
	}

	// Get the current directories directory header
	dvdh, blocks, _, err := dsk.PRODOSFindDirBlocks(2, path)
	if err != nil {
		return err
	}

	// find a freeBlock for the directory
	freeBlocks, err := dsk.PRODOSGetFreeBlocks(1, vdh.GetTotalBlocks())
	if err != nil {
		return err
	}

	// got a file descriptor
	fd.SetStorageType(StorageType_SubDir_File)
	fd.SetAccessMode(AccessType_Default | AccessType_Changed)
	fd.SetCreateTime(time.Now())
	fd.SetModTime(time.Now())
	fd.SetName(name)
	fd.SetType(FileType_PD_Directory)
	fd.SetIndexBlock(freeBlocks[0]) // <------- block for the directory header
	fd.SetTotalBlocks(1)
	fd.SetSize(512)
	fd.SetHeaderPointer(blocks[0])
	fd.SetMinVersion(0x00)
	fd.SetVersion(0x23)

	err = dsk.PRODOSInitDirectoryBlock(freeBlocks[0], blocks[0], dvdh.GetEntriesPerBlock(), dvdh.GetEntryLength(), fd.entryNum, name)
	if err != nil {
		return err
	}

	err = dsk.PRODOSMarkBlocks(freeBlocks, false)
	if err != nil {
		return err
	}

	dvdh.SetFileCount(vdh.GetFileCount() + 1)
	dvdh.Publish(dsk)
	if err != nil {
		return err
	}

	return fd.Publish(dsk)

}

func (dsk *DSKWrapper) PRODOSInitDirectoryBlock(targetBlock int, parentBlock int, entriesPerBlock int, entrySize int, entryNum int, name string) error {

	block := make([]byte, 512)
	dh := &VDH{
		Data:        block[4 : 4+PRODOS_ENTRY_SIZE],
		blockid:     targetBlock,
		blockoffset: 4,
	}

	// Must be set (beneath Apple ProDOS)
	dh.Data[0x10] = 0x75

	dh.SetStorageType(StorageType_SubDir_Header)
	dh.SetName(name)
	dh.SetCreateTime(time.Now())
	dh.SetAccess(AccessType_Default)
	dh.SetEntriesPerBlock(entriesPerBlock)
	dh.SetEntryLength(entrySize)
	dh.SetFileCount(0)
	dh.SetDirParentPointer(parentBlock)
	dh.SetDirParentEntry(entryNum)
	dh.SetDirParentEntryLength(entrySize)
	dh.SetMinVersion(0x00)
	dh.SetVersion(0x23)

	return dsk.PRODOSWrite(targetBlock, block)

}

func (dsk *DSKWrapper) PRODOSDeleteDirectory(path string, name string) error {

	fd, err := dsk.PRODOSGetNamedEntry(path, name)
	if err != nil {
		return err
	}

	// got file descriptor, is it empty
	if fd.GetStorageType() == 0x00 {
		return errors.New("Path not found")
	}

	if fd.Type() != FileType_PD_Directory {
		return errors.New("Not a directory")
	}

	// Get the current directories directory header
	_, files, err := dsk.PRODOSGetCatalog(fd.IndexBlock(), "*")
	if err != nil {
		return err
	}

	// Remove any files in subdirectory
	for _, subfile := range files {
		if subfile.GetStorageType() == StorageType_SubDir_File {
			// FOLDER, RECURSE
			err = dsk.PRODOSDeleteDirectory(path+"/"+name, subfile.NameUnadorned())
			if err != nil {
				return err
			}
		} else if subfile.GetStorageType() == StorageType_Tree {
			// TREE FILE -- UNSUPPORTED FOR NOW
			return errors.New("Tree file handling not supported currently")
		} else {
			err = dsk.PRODOSDeleteFile(path+"/"+name, subfile.NameUnadorned())
			if err != nil {
				return err
			}
		}
	}

	// Check again and make sure it is empty
	_, files, err = dsk.PRODOSGetCatalog(fd.IndexBlock(), "*")
	if err != nil {
		return err
	}
	if len(files) > 0 {
		return errors.New("Could not delete all subfiles")
	}

	// Get blocks that make up the directory
	_, dirBlocks, _, err := dsk.PRODOSFindDirBlocks(2, path+"/"+name)
	if err != nil {
		return err
	}

	// Free all the things!
	err = dsk.PRODOSMarkBlocks(dirBlocks, true)
	if err != nil {
		return err
	}

	// free file entry
	fd.SetStorageType(StorageType_Inactive)
	err = fd.Publish(dsk)
	if err != nil {
		return err
	}

	// reduce count by 1 at top level
	tvdh, _, _, err := dsk.PRODOSFindDirBlocks(2, path)
	if err != nil {
		return err
	}
	tvdh.SetFileCount(tvdh.GetFileCount() - 1)
	return tvdh.Publish(dsk)

}

func (dsk *DSKWrapper) PRODOSSetLocked(path, name string, lock bool) error {

	// We cheat here a bit and use the get first free entry call with
	// autogrow turned off.
	fd, err := dsk.PRODOSGetNamedEntry(path, name)
	if err != nil {
		return err
	}

	fd.SetLocked(lock)
	return fd.Publish(dsk)

}

// PRODOSGetNamedEntry for a given path, will find the file descriptor with name
func (dsk *DSKWrapper) PRODOSGetNamedEntry(path string, name string) (*ProDOSFileDescriptor, error) {

	//fmt.Printf(">>> Path = %s\n", path)

	vdh, blockList, blockData, err := dsk.PRODOSFindDirBlocks(2, path)
	if err != nil {
		return nil, err
	}

	entries := 0

	// if we are here, we found the right directory...
	for idx, data := range blockData {
		count := 0

		if idx == 0 {
			count = 1
			entries = 1
		}

		for count < vdh.GetEntriesPerBlock() {
			offset := 4 + count*PRODOS_ENTRY_SIZE
			chunk := data[offset : offset+PRODOS_ENTRY_SIZE]
			fd := &ProDOSFileDescriptor{}
			fd.SetData(chunk, blockList[idx], offset)
			fd.entryNum = entries

			//fmt.Printf(">>> Comparing %s to %s\n", strings.ToLower(fd.NameUnadorned()), strings.ToLower(name))

			if fd.GetStorageType() != 0x00 && strings.ToLower(fd.NameUnadorned()) == strings.ToLower(name) {
				return fd, nil
			}

			count += 1
			entries++
		}
	}

	return nil, errors.New("Not found")
}

func (dsk *DSKWrapper) PRODOSRenameFile(path, name, newname string) error {

	fd, err := dsk.PRODOSGetNamedEntry(path, name)
	if err != nil {
		return err
	}

	_, err = dsk.PRODOSGetNamedEntry(path, newname)
	if err == nil {
		return errors.New("New name already exists")
	}

	// can rename here
	fd.SetName(newname)
	return fd.Publish(dsk)

}
