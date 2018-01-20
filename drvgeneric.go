package main

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/paleotronic/diskm8/disk"
	"github.com/paleotronic/diskm8/loggy"
)

func analyzeNONE(id int, dsk *disk.DSKWrapper, info *Disk) {

	l := loggy.Get(id)

	// Sector bitmap
	switch len(dsk.Data) {
	case disk.STD_DISK_BYTES:
		info.Tracks = 35
		info.Sectors = 16
	case disk.STD_DISK_BYTES_OLD:
		info.Tracks = 35
		info.Sectors = 13
	case disk.PRODOS_800KB_DISK_BYTES:
		info.Tracks = disk.GetDiskFormat(disk.DF_PRODOS_800KB).TPD()
		info.Sectors = disk.GetDiskFormat(disk.DF_PRODOS_800KB).SPT()
	default:
		l.Errorf("Unknown size %d  bytes", len(dsk.Data))
	}

	l.Logf("Tracks: %d, Sectors: %d", info.Tracks, info.Sectors)

	l.Logf("Reading sector bitmap and SHA256'ing sectors")

	l.Logf("Assuming all sectors might be used")
	info.Bitmap = make([]bool, info.Tracks*info.Sectors)
	for i := range info.Bitmap {
		info.Bitmap[i] = true
	}

	info.ActiveSectors = make(DiskSectors, 0)

	activeData := make([]byte, 0)

	for t := 0; t < info.Tracks; t++ {

		for s := 0; s < info.Sectors; s++ {

			if info.Bitmap[t*info.Sectors+s] {
				sector := &DiskSector{
					Track:  t,
					Sector: s,
					SHA256: dsk.ChecksumSector(t, s),
				}

				data := dsk.Read()
				activeData = append(activeData, data...)

				if *ingestMode&2 == 2 {
					sector.Data = data
				}

				info.ActiveSectors = append(info.ActiveSectors, sector)
			}
		}

	}

	sum := sha256.Sum256(activeData)
	info.SHA256Active = hex.EncodeToString(sum[:])

	info.LogBitmap(id)

	// Analyzing files
	l.Log("Skipping Analysis of files")

	exists := exists(*baseName + "/" + info.GetFilename())

	if !exists || *forceIngest {
		info.WriteToFile(*baseName + "/" + info.GetFilename())
	} else {
		l.Log("Not writing as it already exists")
	}

	out(dsk.Format)

}
