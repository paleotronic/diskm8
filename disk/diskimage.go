package disk

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"crypto/sha256"
	"encoding/hex"

	"fmt"
)

//import "math/rand"

const STD_BYTES_PER_SECTOR = 256
const STD_TRACKS_PER_DISK = 35
const STD_SECTORS_PER_TRACK = 16
const STD_SECTORS_PER_TRACK_OLD = 13
const STD_DISK_BYTES = STD_TRACKS_PER_DISK * STD_SECTORS_PER_TRACK * STD_BYTES_PER_SECTOR
const STD_DISK_BYTES_OLD = STD_TRACKS_PER_DISK * STD_SECTORS_PER_TRACK_OLD * STD_BYTES_PER_SECTOR
const PRODOS_800KB_BLOCKS = 1600
const PRODOS_800KB_DISK_BYTES = STD_BYTES_PER_SECTOR * 2 * PRODOS_800KB_BLOCKS
const PRODOS_400KB_BLOCKS = 800
const PRODOS_400KB_DISK_BYTES = STD_BYTES_PER_SECTOR * 2 * PRODOS_400KB_BLOCKS
const PRODOS_SECTORS_PER_BLOCK = 2
const PRODOS_BLOCKS_PER_TRACK = 8
const PRODOS_800KB_BLOCKS_PER_TRACK = 20
const PRODOS_BLOCKS_PER_DISK = 280
const PRODOS_ENTRY_SIZE = 39

const TRACK_NIBBLE_LENGTH = 0x1A00
const TRACK_COUNT = STD_TRACKS_PER_DISK
const SECTOR_COUNT = STD_SECTORS_PER_TRACK
const HALF_TRACK_COUNT = TRACK_COUNT * 2
const DISK_NIBBLE_LENGTH = TRACK_NIBBLE_LENGTH * TRACK_COUNT
const DISK_PLAIN_LENGTH = STD_DISK_BYTES
const DISK_2MG_NON_NIB_LENGTH = DISK_PLAIN_LENGTH + 0x040
const DISK_2MG_NIB_LENGTH = DISK_NIBBLE_LENGTH + 0x040

type DSKContainer []byte

func Checksum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

type SectorOrder int

var DOS_33_SECTOR_ORDER = []int{
	0x00, 0x07, 0x0E, 0x06, 0x0D, 0x05, 0x0C, 0x04,
	0x0B, 0x03, 0x0A, 0x02, 0x09, 0x01, 0x08, 0x0F,
}

var DOS_32_SECTOR_ORDER = []int{
	0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
	0x08, 0x09, 0x0A, 0x0B, 0x0C,
}

var PRODOS_SECTOR_ORDER = []int{
	0x00, 0x08, 0x01, 0x09, 0x02, 0x0a, 0x03, 0x0b,
	0x04, 0x0c, 0x05, 0x0d, 0x06, 0x0e, 0x07, 0x0f,
}
var LINEAR_SECTOR_ORDER = []int{
	0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
	0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
}

var NIBBLE_62 = []byte{
	0x96, 0x97, 0x9a, 0x9b, 0x9d, 0x9e, 0x9f, 0xa6,
	0xa7, 0xab, 0xac, 0xad, 0xae, 0xaf, 0xb2, 0xb3,
	0xb4, 0xb5, 0xb6, 0xb7, 0xb9, 0xba, 0xbb, 0xbc,
	0xbd, 0xbe, 0xbf, 0xcb, 0xcd, 0xce, 0xcf, 0xd3,
	0xd6, 0xd7, 0xd9, 0xda, 0xdb, 0xdc, 0xdd, 0xde,
	0xdf, 0xe5, 0xe6, 0xe7, 0xe9, 0xea, 0xeb, 0xec,
	0xed, 0xee, 0xef, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6,
	0xf7, 0xf9, 0xfa, 0xfb, 0xfc, 0xfd, 0xfe, 0xff}

var NIBBLE_53 = []byte{
	0xab, 0xad, 0xae, 0xaf, 0xb5, 0xb6, 0xb7, 0xba,
	0xbb, 0xbd, 0xbe, 0xbf, 0xd6, 0xd7, 0xda, 0xdb,
	0xdd, 0xde, 0xdf, 0xea, 0xeb, 0xed, 0xee, 0xef,
	0xf5, 0xf6, 0xf7, 0xfa, 0xfb, 0xfd, 0xfe, 0xff,
}

var identity = map[string]DiskFormat{
	"99b900080a0a0a990008c8d0f4a62ba9098527adcc03854184408a4a4a4a4a09": GetDiskFormat(DF_DOS_SECTORS_13),
	"01a527c909d018a52b4a4a4a4a09c0853fa95c853e18adfe086dff088dfe08ae": GetDiskFormat(DF_DOS_SECTORS_16),
	"0138b0034c32a18643c903088a29704a4a4a4a09c08549a0ff844828c8b148d0": GetDiskFormat(DF_PRODOS),
}

const (
	SectorOrderDOS33 SectorOrder = iota
	SectorOrderDOS32
	SectorOrderDOS33Alt
	SectorOrderProDOS
	SectorOrderProDOSLinear
)

func (so SectorOrder) String() string {
	switch so {
	case SectorOrderDOS32:
		return "DOS"
	case SectorOrderDOS33:
		return "DOS"
	case SectorOrderDOS33Alt:
		return "DOS Alternate"
	case SectorOrderProDOS:
		return "ProDOS"
	case SectorOrderProDOSLinear:
		return "Linear"
	}

	return "Linear"
}

type DiskFormatID int

const (
	DF_NONE DiskFormatID = iota
	DF_DOS_SECTORS_13
	DF_DOS_SECTORS_16
	DF_PRODOS
	DF_PRODOS_800KB
	DF_PASCAL
	DF_RDOS_3
	DF_RDOS_32
	DF_RDOS_33
	DF_PRODOS_400KB
	DF_PRODOS_CUSTOM
)

type DiskFormat struct {
	ID   DiskFormatID
	bpd  int
	spt  int
	uspt int
	tpd  int
}

func GetDiskFormat(id DiskFormatID) DiskFormat {
	return DiskFormat{ID: id}
}

func GetPDDiskFormat(id DiskFormatID, blocks int) DiskFormat {
	return DiskFormat{
		ID:   id,
		bpd:  blocks,
		tpd:  80,
		spt:  blocks / 80,
		uspt: blocks / 80,
	}
}

func (f DiskFormat) String() string {
	switch f.ID {
	case DF_NONE:
		return "Unrecognized"
	case DF_DOS_SECTORS_13:
		return "Apple DOS 13 Sector"
	case DF_DOS_SECTORS_16:
		return "Apple DOS 16 Sector"
	case DF_PRODOS:
		return "ProDOS"
	case DF_PASCAL:
		return "Pascal"
	case DF_PRODOS_400KB:
		return "ProDOS 400Kb"
	case DF_PRODOS_800KB:
		return "ProDOS 800Kb"
	case DF_RDOS_3:
		return "SSI RDOS 3 (16/13/Physical)"
	case DF_RDOS_32:
		return "SSI RDOS 32 (13/13/Physical)"
	case DF_RDOS_33:
		return "SSI RDOS 32 (16/16/PD)"
	case DF_PRODOS_CUSTOM:
		return fmt.Sprintf("ProDOS Custom (%d SPT, %d TPD)", f.SPT(), f.TPD())
	}
	return "Unrecognized"
}

func (df DiskFormat) BPD() int {
	switch df.ID {
	case DF_RDOS_3:
		return 222
	case DF_RDOS_32:
		return 222
	case DF_RDOS_33:
		return 280
	case DF_DOS_SECTORS_13:
		return 222
	case DF_DOS_SECTORS_16:
		return 280
	case DF_PRODOS:
		return 280
	case DF_PASCAL:
		return 280
	case DF_PRODOS_800KB:
		return 1600
	case DF_PRODOS_400KB:
		return 800
	case DF_PRODOS_CUSTOM:
		return df.bpd
	}
	return 16 // fallback
}

func (df DiskFormat) USPT() int {
	switch df.ID {
	case DF_RDOS_3:
		return 13
	case DF_RDOS_32:
		return 13
	case DF_RDOS_33:
		return 16
	case DF_DOS_SECTORS_13:
		return 13
	case DF_DOS_SECTORS_16:
		return 16
	case DF_PRODOS:
		return 16
	case DF_PASCAL:
		return 16
	case DF_PRODOS_800KB:
		return 40
	case DF_PRODOS_400KB:
		return 20
	case DF_PRODOS_CUSTOM:
		return df.uspt
	}
	return 16 // fallback
}

func (df DiskFormat) SPT() int {
	switch df.ID {
	case DF_RDOS_3:
		return 16
	case DF_RDOS_32:
		return 13
	case DF_RDOS_33:
		return 16
	case DF_DOS_SECTORS_13:
		return 13
	case DF_DOS_SECTORS_16:
		return 16
	case DF_PRODOS:
		return 16
	case DF_PASCAL:
		return 16
	case DF_PRODOS_800KB:
		return 40
	case DF_PRODOS_400KB:
		return 20
	case DF_PRODOS_CUSTOM:
		return df.spt
	}
	return 16 // fallback
}

func (df DiskFormat) TPD() int {
	switch df.ID {
	case DF_RDOS_3:
		return 35
	case DF_RDOS_32:
		return 35
	case DF_RDOS_33:
		return 35
	case DF_DOS_SECTORS_13:
		return 35
	case DF_DOS_SECTORS_16:
		return 35
	case DF_PRODOS:
		return 35
	case DF_PASCAL:
		return 35
	case DF_PRODOS_800KB:
		return 80
	case DF_PRODOS_400KB:
		return 80
	case DF_PRODOS_CUSTOM:
		return df.tpd
	}
	return 35 // fallback
}

type Nibbler interface {
	SetNibble(offset int, value byte)
	GetNibble(offset int) byte
}

type DSKWrapper struct {
	Data          DSKContainer
	Layout        SectorOrder
	CurrentTrack  int
	CurrentSector int
	SectorPointer int
	Format        DiskFormat
	RDOSFormat    RDOSFormat
	Filename      string
	//Nibbles       []byte
	Nibbles            Nibbler
	CurrentSectorOrder []int
	WriteProtected     bool
	NibblesChanged     bool
}

// SectoreMapperDOS33 handles the interleaving for dos sectors
func SectorMapperDOS33(wanted int) int {

	return wanted

	switch wanted {
	case 0:
		return 0
	case 13:
		return 1
	case 11:
		return 2
	case 9:
		return 3
	case 7:
		return 4
	case 5:
		return 5
	case 3:
		return 6
	case 1:
		return 7
	case 14:
		return 8
	case 12:
		return 9
	case 10:
		return 10
	case 8:
		return 11
	case 6:
		return 12
	case 4:
		return 13
	case 2:
		return 14
	case 15:
		return 15
	}

	return -1 // invalid sector
}

func SectorMapperDOS33Alt(wanted int) int {

	switch wanted {
	case 0:
		return 0
	case 13:
		return 1
	case 11:
		return 2
	case 9:
		return 3
	case 7:
		return 4
	case 5:
		return 5
	case 3:
		return 6
	case 1:
		return 7
	case 14:
		return 8
	case 12:
		return 9
	case 10:
		return 10
	case 8:
		return 11
	case 6:
		return 12
	case 4:
		return 13
	case 2:
		return 14
	case 15:
		return 15
	}

	return -1 // invalid sector
}

func SectorMapperDOS33bad(wanted int) int {

	return wanted

	switch wanted {
	case 0:
		return 0
	case 1:
		return 13
	case 2:
		return 11
	case 3:
		return 9
	case 4:
		return 7
	case 5:
		return 5
	case 6:
		return 3
	case 7:
		return 1
	case 8:
		return 14
	case 9:
		return 12
	case 10:
		return 10
	case 11:
		return 8
	case 12:
		return 6
	case 13:
		return 4
	case 14:
		return 2
	case 15:
		return 15
	}

	return -1 // invalid sector
}

// SectoreMapperProDOS handles the interleaving for dos sectors
func SectorMapperProDOS(wanted int) int {
	switch wanted {
	case 0:
		return 0
	case 1:
		return 2
	case 2:
		return 4
	case 3:
		return 6
	case 4:
		return 8
	case 5:
		return 10
	case 6:
		return 12
	case 7:
		return 14
	case 8:
		return 1
	case 9:
		return 3
	case 10:
		return 5
	case 11:
		return 7
	case 12:
		return 9
	case 13:
		return 11
	case 14:
		return 13
	case 15:
		return 15
	}

	return -1 // invalid sector
}

func (d *DSKWrapper) SetTrack(t int) error {

	if t >= 0 && t < d.Format.TPD() {
		d.CurrentTrack = t
		d.SetSectorPointer()
		return nil
	}

	return errors.New("Invalid track")

}

// SetSector changes the sector we are looking at
func (d *DSKWrapper) SetSector(s int) error {
	if s >= 0 && s < d.Format.USPT() {
		d.CurrentSector = s
		d.SetSectorPointer()
		return nil
	}

	return errors.New("Invalid sector")
}

func (d *DSKWrapper) HuntVTOC(t, s int) (int, int) {
	for block := 0; block < len(d.Data)/256; block++ {
		data := d.Data[block*256 : block*256+256]
		var v VTOC
		v.SetData(data, (block / s), (block % s))
		if v.GetTracks() == t && v.GetSectors() == s {
			return (block / s), (block % s)
		}
	}
	return -1, -1
}

// SetSectorPointer calculates the pointer to the current sector, taking into
// account the sector interleaving of the DSK image.
func (d *DSKWrapper) SetSectorPointer() {

	track := d.CurrentTrack
	sector := d.CurrentSector
	isector := sector
	switch d.Layout {
	case SectorOrderDOS33Alt:
		isector = SectorMapperDOS33Alt(sector)
	case SectorOrderDOS33:
		isector = SectorMapperDOS33(sector)
	case SectorOrderProDOS:
		isector = SectorMapperProDOS(sector)
	}

	d.SectorPointer = (track * d.Format.SPT() * STD_BYTES_PER_SECTOR) + (STD_BYTES_PER_SECTOR * isector)
}

func (d *DSKWrapper) UpdateTrack(track int) {
	d.NibblesChanged = true
}

func (d *DSKWrapper) ChecksumDisk() string {
	return Checksum(d.Data)
}

func (d *DSKWrapper) ChecksumSector(t, s int) string {
	d.SetTrack(t)
	d.SetSector(s)
	return Checksum(d.Data[d.SectorPointer : d.SectorPointer+256])
}

func (d *DSKWrapper) IsChanged() bool {
	return d.NibblesChanged
}

// func (d *DSKWrapper) GetNibbles() []byte {
// 	return d.Nibbles
// }

// Read is a simple function to return the current pointed to sector
func (d *DSKWrapper) Read() []byte {
	////fmt.Printf("---> Reading track %d, sector %d\n", d.CurrentTrack, d.CurrentSector)
	return d.Data[d.SectorPointer : d.SectorPointer+256]
}

func (d *DSKWrapper) Write(data []byte) {
	////fmt.Printf("---> Reading track %d, sector %d\n", d.CurrentTrack, d.CurrentSector)
	l := len(data)
	if l > STD_BYTES_PER_SECTOR {
		l = STD_BYTES_PER_SECTOR
	}

	for i, v := range data {
		if i >= l {
			break
		}
		d.Data[d.SectorPointer+i] = v
	}
}

// Seek is a convienience function to go straight to a particular track & sector
func (d *DSKWrapper) Seek(t, s int) error {

	var e error

	e = d.SetTrack(t)
	if e != nil {
		return e
	}

	e = d.SetSector(s)

	return e
}

func (d *DSKWrapper) SetData(data []byte) {
	//	for i, v := range data {
	//		d.Data[i] = v
	//	}
	d.Data = data
}

func NewDSKWrapper(nibbler Nibbler, filename string) (*DSKWrapper, error) {

	f, e := os.Open(filename)
	if e != nil {
		return nil, e
	}
	data, e := ioutil.ReadAll(f)
	f.Close()
	if e != nil {
		return nil, e
	}

	w, e := NewDSKWrapperBin(nibbler, data, filename)
	return w, e

}

func NewDSKWrapperBin(nibbler Nibbler, data []byte, filename string) (*DSKWrapper, error) {

	if len(data) != 232960 &&
		len(data) != STD_DISK_BYTES &&
		len(data) != STD_DISK_BYTES_OLD &&
		len(data) != PRODOS_400KB_DISK_BYTES &&
		len(data) != PRODOS_400KB_DISK_BYTES+64 &&
		len(data) != PRODOS_800KB_DISK_BYTES &&
		len(data) != PRODOS_800KB_DISK_BYTES+64 &&
		len(data) != STD_DISK_BYTES+64 {
		return nil, errors.New("Incorrect disk bytes")
	}

	this := &DSKWrapper{}

	this.SetData(data)
	this.Filename = filename
	this.Layout = SectorOrderDOS33
	this.CurrentSectorOrder = DOS_33_SECTOR_ORDER
	this.Nibbles = nibbler
	this.WriteProtected = false

	this.Identify()

	return this, nil

}

func (dsk *DSKWrapper) GetNibbles() []byte {

	n := make([]byte, DISK_NIBBLE_LENGTH)
	for i, _ := range n {
		n[i] = dsk.Nibbles.GetNibble(i)
	}

	return n

}

func (dsk *DSKWrapper) SetNibbles(data []byte) {
	if dsk.Nibbles == nil {
		return
	}
	for i, v := range data {
		dsk.Nibbles.SetNibble(i, v)
	}
}

func (dsk *DSKWrapper) Identify() {

	dsk.Format = GetDiskFormat(DF_NONE)

	var hint DiskFormat

	dsk.Filename = strings.ToLower(dsk.Filename)

	switch {
	case strings.HasSuffix(dsk.Filename, ".po"):
		hint = GetDiskFormat(DF_PRODOS)
	case strings.HasSuffix(dsk.Filename, ".do"):
		hint = GetDiskFormat(DF_DOS_SECTORS_16)
	default:
		hint = GetDiskFormat(DF_DOS_SECTORS_16)
	}

	is2MG, Format, Layout, zdsk := dsk.Is2MG()
	if is2MG {
		////fmt.Println("repacked", len(zdsk.Data))
		dsk.SetData(zdsk.Data)
		dsk.Layout = Layout
		dsk.Format = Format
		return
	}

	isPD, Format, Layout := dsk.IsProDOS()
	if isPD {
		if Format == GetDiskFormat(DF_PRODOS) {
			dsk.Format = GetDiskFormat(DF_PRODOS)
			dsk.Layout = Layout
			switch dsk.Layout {
			case SectorOrderProDOS:
				dsk.CurrentSectorOrder = PRODOS_SECTOR_ORDER
			case SectorOrderProDOSLinear:
				dsk.CurrentSectorOrder = PRODOS_SECTOR_ORDER
			case SectorOrderDOS33:
				dsk.CurrentSectorOrder = DOS_33_SECTOR_ORDER
			}
			dsk.SetNibbles(dsk.Nibblize())
			return
		} else {
			dsk.Format = GetDiskFormat(DF_PRODOS_800KB)
			dsk.Layout = Layout
			switch dsk.Layout {
			case SectorOrderProDOS:
				dsk.CurrentSectorOrder = PRODOS_SECTOR_ORDER
			case SectorOrderProDOSLinear:
				dsk.CurrentSectorOrder = PRODOS_SECTOR_ORDER
			case SectorOrderDOS33:
				dsk.CurrentSectorOrder = DOS_33_SECTOR_ORDER
			}
			dsk.SetNibbles(make([]byte, 232960))
			return
		}
	}

	isRDOS, Version := dsk.IsRDOS()
	if isRDOS {
		dsk.RDOSFormat = Version
		switch Version {
		case RDOS_3:
			dsk.Format = GetDiskFormat(DF_RDOS_3)
			dsk.Layout = SectorOrderDOS33Alt
			dsk.CurrentSectorOrder = DOS_33_SECTOR_ORDER
			dsk.SetNibbles(dsk.Nibblize())
		case RDOS_32:
			dsk.Format = GetDiskFormat(DF_RDOS_32)
			dsk.Layout = SectorOrderDOS33Alt
			dsk.CurrentSectorOrder = DOS_33_SECTOR_ORDER
			dsk.SetNibbles(make([]byte, 232960)) // FIXME: fix nibbles
		case RDOS_33:
			dsk.Format = GetDiskFormat(DF_RDOS_33)
			dsk.Layout = SectorOrderProDOS
			dsk.CurrentSectorOrder = PRODOS_SECTOR_ORDER
			dsk.SetNibbles(dsk.Nibblize())
		}
		return
	}

	isAppleDOS, Format, Layout := dsk.IsAppleDOS()
	if isAppleDOS {
		////fmt.Printf("Format: %s\n", Format.String())
		dsk.Format = Format
		dsk.Layout = Layout
		switch Layout {
		case SectorOrderProDOS:
			////fmt.Println("Sector Order: ProDOS")
			dsk.CurrentSectorOrder = PRODOS_SECTOR_ORDER
		case SectorOrderProDOSLinear:
			////fmt.Println("Sector Order: Linear")
			dsk.CurrentSectorOrder = LINEAR_SECTOR_ORDER
		case SectorOrderDOS33:
			////fmt.Println("Sector Order: DOS33")
			dsk.CurrentSectorOrder = DOS_33_SECTOR_ORDER
		case SectorOrderDOS32:
			////fmt.Println("Sector Order: DOS32")
			//dsk.CurrentSectorOrder = DOS_32_SECTOR_ORDER
		case SectorOrderDOS33Alt:
			////fmt.Println("Sector Order: Alt Linear")
			dsk.CurrentSectorOrder = LINEAR_SECTOR_ORDER
		}
		dsk.SetNibbles(dsk.Nibblize())
		return
	}

	fp := hex.EncodeToString(dsk.Data[:32])
	if dfmt, ok := identity[fp]; ok {
		dsk.Format = dfmt
		//fmt.Println(dsk.Format.String())
	}

	//fmt.Printf("Disk name: %s\n", dsk.Filename)

	// 1. NIB
	if len(dsk.Data) == 232960 {
		//fmt.Println("THIS IS A NIB")
		dsk.Format = GetDiskFormat(DF_DOS_SECTORS_16)
		dsk.SetNibbles(dsk.Data)
		return
	}

	// 2. Wrong size
	if len(dsk.Data) != STD_DISK_BYTES && len(dsk.Data) != STD_DISK_BYTES_OLD && len(dsk.Data) != PRODOS_800KB_DISK_BYTES {
		//fmt.Println("NOT STANDARD DISK")
		dsk.Format = GetDiskFormat(DF_NONE)
		dsk.SetNibbles(make([]byte, 232960))
		return
	}

	// 3. DOS 3x Disk
	vtoc, e := dsk.AppleDOSGetVTOC()
	//fmt.Println(vtoc.GetTracks(), vtoc.GetSectors())
	if e == nil && vtoc.GetTracks() == 35 {
		//bps := vtoc.BytesPerSector()
		t := vtoc.GetTracks()
		s := vtoc.GetSectors()

		//fmt.Printf("DOS Tracks = %d, Sectors = %d\n", t, s)

		if t == 35 && s == 16 {
			dsk.Layout = SectorOrderDOS33
			dsk.CurrentSectorOrder = DOS_33_SECTOR_ORDER
			dsk.Format = GetDiskFormat(DF_DOS_SECTORS_16)
			dsk.SetNibbles(dsk.Nibblize())
		} else if t == 35 && s == 13 {
			dsk.Layout = SectorOrderDOS32
			dsk.CurrentSectorOrder = DOS_32_SECTOR_ORDER
			dsk.Format = GetDiskFormat(DF_DOS_SECTORS_13)
			dsk.SetNibbles(make([]byte, 232960))
		}
		return
	}

	//fmt.Println("Trying prodos / pascal")

	isPAS, volName := dsk.IsPascal()
	if isPAS && volName != "" {
		dsk.Format = GetDiskFormat(DF_PASCAL)
		dsk.Layout = SectorOrderDOS33
		dsk.CurrentSectorOrder = DOS_33_SECTOR_ORDER
		dsk.SetNibbles(dsk.Nibblize())
		return
	}

	dsk.CurrentSectorOrder = PRODOS_SECTOR_ORDER

	if dsk.Format == GetDiskFormat(DF_PRODOS) && len(dsk.Data) == PRODOS_800KB_DISK_BYTES {
		dsk.Format = GetDiskFormat(DF_PRODOS_800KB)
	}

	//fmt.Println(dsk.Format.String())

	switch dsk.Format.ID {

	case DF_PRODOS:

		vdh, e := dsk.PRODOSGetVDH(2)
		////fmt.Printf("Blocks = %d\n", vdh.GetTotalBlocks())
		if e == nil && vdh.GetStorageType() == 0xf && vdh.GetTotalBlocks() == 280 {
			dsk.Format = GetDiskFormat(DF_PRODOS)
			dsk.Layout = SectorOrderDOS33
			dsk.CurrentSectorOrder = DOS_33_SECTOR_ORDER
			dsk.SetNibbles(dsk.Nibblize())

			//fmt.Println("THIS IS A PRODOS DISKETTE, DOS Ordered")
			return
		} else {
			dsk.Format = GetDiskFormat(DF_PRODOS)
			dsk.Layout = SectorOrderDOS33Alt

			////fmt.Println("Try again")

			vdh, e = dsk.PRODOSGetVDH(2)

			if e == nil && vdh.GetStorageType() == 0xf && vdh.GetTotalBlocks() == 280 {

				dsk.CurrentSectorOrder = PRODOS_SECTOR_ORDER
				dsk.SetNibbles(dsk.Nibblize())

				//fmt.Println("THIS IS A PRODOS DISKETTE, ProDOS Ordered")
				return

			}
		}

	case DF_PRODOS_800KB:

		vdh, e := dsk.PRODOS800GetVDH(2)
		//fmt.Printf("Blocks = %d\n", vdh.GetTotalBlocks())
		if e == nil && vdh.GetStorageType() == 0xf && vdh.GetTotalBlocks() == 1600 {
			dsk.Format = GetDiskFormat(DF_PRODOS_800KB)
			dsk.Layout = SectorOrderDOS33
			dsk.CurrentSectorOrder = DOS_33_SECTOR_ORDER
			//dsk.SetNibbles(dsk.nibblize())
			dsk.SetNibbles(make([]byte, 232960))

			//fmt.Println("THIS IS A PRODOS DISKETTE, DOS Ordered")
			return
		}

	}

	switch hint.ID {
	case DF_PRODOS:
		dsk.Format = GetDiskFormat(DF_PRODOS)
		dsk.Layout = SectorOrderProDOS
		dsk.CurrentSectorOrder = PRODOS_SECTOR_ORDER
		dsk.SetNibbles(dsk.Nibblize())
		//fmt.Println("VTOC read failed, will nibblize anyway...")
	case DF_DOS_SECTORS_16:
		//fmt.Println("VTOC read failed, will nibblize anyway...")
		dsk.Layout = SectorOrderProDOS
		dsk.CurrentSectorOrder = DOS_33_SECTOR_ORDER
		dsk.Format = GetDiskFormat(DF_DOS_SECTORS_16)
		dsk.SetNibbles(dsk.Nibblize())
	}

}

//func (d *DSKWrapper) ReadFileSectorsProDOS(fd ProDOSFileDescriptor) ([]byte, error) {

//}

func Dump(bytes []byte) {
	perline := 0xC
	base := 0
	ascii := ""
	for i, v := range bytes {
		if i%perline == 0 {
			fmt.Println(" " + ascii)
			ascii = ""
			fmt.Printf("%.4X:", base+i)
		}
		if v >= 32 && v < 128 {
			ascii += string(rune(v))
		} else {
			ascii += "."
		}
		fmt.Printf(" %.2X", v)
	}
	fmt.Println(" " + ascii)
}

func Between(v, lo, hi uint) bool {
	return ((v >= lo) && (v <= hi))
}

func PokeToAscii(v uint, usealt bool) int {
	highbit := v & 1024

	v = v & 1023

	if Between(v, 0, 31) {
		return int((64 + (v % 32)) | highbit)
	}

	if Between(v, 32, 63) {
		return int((32 + (v % 32)) | highbit)
	}

	if Between(v, 64, 95) {
		if usealt {
			return int((128 + (v % 32)) | highbit)
		} else {
			return int((64 + (v % 32)) | highbit)
		}
	}

	if Between(v, 96, 127) {
		if usealt {
			return int((96 + (v % 32)) | highbit)
		} else {
			return int((32 + (v % 32)) | highbit)
		}
	}

	if Between(v, 128, 159) {
		return int((64 + (v % 32)) | highbit)
	}

	if Between(v, 160, 191) {
		return int((32 + (v % 32)) | highbit)
	}

	if Between(v, 192, 223) {
		return int((64 + (v % 32)) | highbit)
	}

	if Between(v, 224, 255) {
		return int((96 + (v % 32)) | highbit)
	}

	return int(v | highbit)
}

func (d *DSKWrapper) Nibblize() []byte {

	if len(d.Data) != STD_DISK_BYTES {
		return make([]byte, 232960)
	}

	data := d.Data

	output := bytes.NewBuffer([]byte(nil))

	for track := 0; track < STD_TRACKS_PER_DISK; track++ {
		//d.writeJunkBytes(output, 48);
		for sector := 0; sector < STD_SECTORS_PER_TRACK; sector++ {
			//gap2 := int((rand.Float32() * 5.0) + 4)
			gap2 := 6
			// 15 junk bytes
			d.writeJunkBytes(output, 15)
			// Address block
			d.writeAddressBlock(output, track, sector, 254)
			// 4 junk bytes
			d.writeJunkBytes(output, gap2)
			// Data block
			d.nibblizeBlock(output, track, d.CurrentSectorOrder[sector], data)
			// 34 junk bytes
			d.writeJunkBytes(output, 38-gap2)
		}
	}

	return output.Bytes()

}

func (d *DSKWrapper) NibbleOffsetToTS(offset int) (int, int) {
	offset = offset - (offset % 256)
	c := offset / 256
	sector := c % SECTOR_COUNT
	track := (c - sector) / SECTOR_COUNT
	return track, sector
}

func (d *DSKWrapper) nibblizeBlock(output io.Writer, track, sector int, nibbles []byte) {

	//log.Printf("NibblizeBlock(%d, %d)", track, sector)

	offset := ((track * SECTOR_COUNT) + sector) * 256
	temp := make([]int, 342)
	for i := 0; i < 256; i++ {
		temp[i] = int((nibbles[offset+i] & 0x0ff) >> 2)
	}
	hi := 0x001
	med := 0x0AB
	low := 0x055

	for i := 0; i < 0x56; i++ {
		value := ((nibbles[offset+hi] & 1) << 5) |
			((nibbles[offset+hi] & 2) << 3) |
			((nibbles[offset+med] & 1) << 3) |
			((nibbles[offset+med] & 2) << 1) |
			((nibbles[offset+low] & 1) << 1) |
			((nibbles[offset+low] & 2) >> 1)
		temp[i+256] = int(value)
		hi = (hi - 1) & 0x0ff
		med = (med - 1) & 0x0ff
		low = (low - 1) & 0x0ff
	}
	output.Write([]byte{0x0d5, 0x0aa, 0x0ad})

	last := 0
	for i := len(temp) - 1; i > 255; i-- {
		value := temp[i] ^ last
		output.Write([]byte{NIBBLE_62[value]})
		last = temp[i]
	}
	for i := 0; i < 256; i++ {
		value := temp[i] ^ last
		output.Write([]byte{NIBBLE_62[value]})
		last = temp[i]
	}
	// Last data byte used as checksum
	output.Write([]byte{NIBBLE_62[last]})
	output.Write([]byte{0x0de, 0x0aa, 0x0eb})

}

func (d *DSKWrapper) writeJunkBytes(output io.Writer, i int) {
	for c := 0; c < i; c++ {
		output.Write([]byte{0xff})
	}
}

func (d *DSKWrapper) writeAddressBlock(output io.Writer, track, sector int, volumeNumber int) {
	output.Write([]byte{0x0d5, 0x0aa, 0x096})

	var checksum int = 0x00
	// volume
	checksum ^= volumeNumber
	output.Write(d.getOddEven(volumeNumber))
	// track
	checksum ^= track
	output.Write(d.getOddEven(track))
	// sector
	checksum ^= sector
	output.Write(d.getOddEven(sector))
	// checksum
	output.Write(d.getOddEven(checksum & 0x0ff))

	output.Write([]byte{0xde, 0xaa, 0xeb})
}

func (d *DSKWrapper) getOddEven(i int) []byte {
	out := []byte{0, 0}
	out[0] = byte(0xAA | (i >> 1))
	out[1] = byte(0xAA | i)
	return out
}
