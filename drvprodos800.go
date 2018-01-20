package main

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/paleotronic/diskm8/disk"
	"github.com/paleotronic/diskm8/loggy"
)

func analyzePRODOS800(id int, dsk *disk.DSKWrapper, info *Disk) {

	l := loggy.Get(id)

	// Sector bitmap
	l.Logf("Reading Disk VTOC...")
	vtoc, err := dsk.PRODOS800GetVDH(2)
	if err != nil {
		l.Errorf("Error reading VTOC: %s", err.Error())
		return
	}

	info.Blocks = vtoc.GetTotalBlocks()
	l.Logf("Blocks: %d", info.Blocks)

	l.Logf("Reading sector bitmap and SHA256'ing sectors")

	info.Bitmap = make([]bool, info.Blocks)

	info.ActiveSectors = make(DiskSectors, 0)

	activeData := make([]byte, 0)

	vbitmap, err := dsk.PRODOS800GetVolumeBitmap()
	if err != nil {
		l.Errorf("Error reading volume bitmap: %s", err.Error())
		return
	}

	l.Debug(vbitmap)

	for b := 0; b < info.Blocks; b++ {
		info.Bitmap[b] = !vbitmap.IsBlockFree(b)

		if info.Bitmap[b] {

			data, _ := dsk.PRODOS800GetBlock(b)

			t, s1, s2 := dsk.PRODOS800GetBlockSectors(b)

			sec1 := &DiskSector{
				Track:  t,
				Sector: s1,
				SHA256: dsk.ChecksumSector(t, s1),
			}

			sec2 := &DiskSector{
				Track:  t,
				Sector: s2,
				SHA256: dsk.ChecksumSector(t, s2),
			}

			if *ingestMode&2 == 2 {
				sec1.Data = data[:256]
				sec2.Data = data[256:]
			}

			info.ActiveSectors = append(info.ActiveSectors, sec1, sec2)

			activeData = append(activeData, data...)

		} else {

			data, _ := dsk.PRODOS800GetBlock(b)

			t, s1, s2 := dsk.PRODOS800GetBlockSectors(b)

			sec1 := &DiskSector{
				Track:  t,
				Sector: s1,
				SHA256: dsk.ChecksumSector(t, s1),
			}

			sec2 := &DiskSector{
				Track:  t,
				Sector: s2,
				SHA256: dsk.ChecksumSector(t, s2),
			}

			if *ingestMode&2 == 2 {
				sec1.Data = data[:256]
				sec2.Data = data[256:]
			}

			info.InactiveSectors = append(info.InactiveSectors, sec1, sec2)

			//activeData = append(activeData, data...)

		}

	}

	sum := sha256.Sum256(activeData)
	info.SHA256Active = hex.EncodeToString(sum[:])

	info.LogBitmap(id)

	// // Analyzing files
	l.Log("Starting Analysis of files")

	info.Files = make([]*DiskFile, 0)
	prodosDir(id, 2, "", dsk, info)

	exists := exists(*baseName + "/" + info.GetFilename())

	if !exists || *forceIngest {
		info.WriteToFile(*baseName + "/" + info.GetFilename())
	} else {
		l.Log("Not writing as it already exists")
	}

	out(dsk.Format)

}
