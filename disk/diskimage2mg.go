package disk

import "fmt"

/*
	2MG format loader...
*/

const PREAMBLE_2MG_SIZE = 0x40

var MAGIC_2MG = []byte{byte('2'), byte('I'), byte('M'), byte('G')}

type Header2MG struct {
	Data [64]byte
}

func (h *Header2MG) SetData(data []byte) {
	for i, v := range data {
		if i < 64 {
			h.Data[i] = v
		}
	}
}

func (h *Header2MG) GetID() string {
	return string(h.Data[0x00:0x04])
}

func (h *Header2MG) GetCreatorID() string {
	return string(h.Data[0x04:0x08])
}

func (h *Header2MG) GetHeaderSize() int {
	return int(h.Data[0x08]) + 256*int(h.Data[0x09])
}

func (h *Header2MG) GetVersion() int {
	return int(h.Data[0x0A]) + 256*int(h.Data[0x0B])
}

func (h *Header2MG) GetImageFormat() int {
	return int(h.Data[0x0C]) + 256*int(h.Data[0x0D]) + 65336*int(h.Data[0x0E]) + 16777216*int(h.Data[0x0F])
}

func (h *Header2MG) GetDOSFlags() int {
	return int(h.Data[0x10]) + 256*int(h.Data[0x11]) + 65336*int(h.Data[0x12]) + 16777216*int(h.Data[0x13])
}

func (h *Header2MG) GetProDOSBlocks() int {
	return int(h.Data[0x14]) + 256*int(h.Data[0x15]) + 65336*int(h.Data[0x16]) + 16777216*int(h.Data[0x17])
}

func (h *Header2MG) GetDiskDataStart() int {
	return int(h.Data[0x18]) + 256*int(h.Data[0x19]) + 65336*int(h.Data[0x1A]) + 16777216*int(h.Data[0x1B])
}

func (h *Header2MG) GetDiskDataLength() int {
	return int(h.Data[0x1C]) + 256*int(h.Data[0x1D]) + 65336*int(h.Data[0x1E]) + 16777216*int(h.Data[0x1F])
}

func (dsk *DSKWrapper) Is2MG() (bool, DiskFormat, SectorOrder, *DSKWrapper) {

	h := &Header2MG{}
	h.SetData(dsk.Data[:0x40])

	if h.GetID() != "2IMG" {
		return false, GetDiskFormat(DF_NONE), SectorOrderDOS33, nil
	}

	fmt.Println("Disk has 2MG Magic")
	fmt.Printf("Block count %d\n", h.GetProDOSBlocks())

	start := h.GetDiskDataStart()
	size := h.GetDiskDataLength()

	if size < len(dsk.Data)-start {
		size = len(dsk.Data) - start
	}

	if size != STD_DISK_BYTES && size != PRODOS_800KB_DISK_BYTES && size != PRODOS_400KB_DISK_BYTES {
		fmt.Printf("Bad size %d bytes @ start %d\n", size, start)
		return false, GetDiskFormat(DF_NONE), SectorOrderDOS33, nil
	}

	data := dsk.Data[start : start+size]
	format := h.GetImageFormat()
	switch format {
	case 0x00: /* DOS sector order */
		zdsk, _ := NewDSKWrapperBin(dsk.Nibbles, data, dsk.Filename)
		return true, GetDiskFormat(DF_DOS_SECTORS_16), SectorOrderDOS33, zdsk
	case 0x01: /* ProDOS sector order */
		zdsk, _ := NewDSKWrapperBin(dsk.Nibbles, data, dsk.Filename)

		if h.GetProDOSBlocks() == 1600 {
			return true, GetDiskFormat(DF_PRODOS_800KB), SectorOrderProDOSLinear, zdsk
		} else if h.GetProDOSBlocks() == 800 {
			return true, GetDiskFormat(DF_PRODOS_400KB), SectorOrderProDOSLinear, zdsk
		} else {
			return true, GetPDDiskFormat(DF_PRODOS_CUSTOM, h.GetProDOSBlocks()), SectorOrderProDOSLinear, zdsk
		}

	}

	return false, GetDiskFormat(DF_NONE), SectorOrderDOS33, nil

}
