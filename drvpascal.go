package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/paleotronic/diskm8/disk"
	"github.com/paleotronic/diskm8/loggy"
)

func dump(in []byte) string {
	out := ""
	for i, v := range in {
		if i%16 == 0 {
			if out != "" {
				out += "\n"
			}
			out += fmt.Sprintf("%.4x: ", i)
		}
		out += fmt.Sprintf("%.2x ", v)
	}
	out += "\n"
	return out
}

func analyzePASCAL(id int, dsk *disk.DSKWrapper, info *Disk) {

	l := loggy.Get(id)

	// Sector bitmap
	l.Logf("Reading Disk Structure...")

	info.Blocks = dsk.Format.BPD()

	l.Logf("Blocks: %d", info.Blocks)

	l.Logf("Reading sector bitmap and SHA256'ing sectors")

	info.Bitmap = make([]bool, info.Tracks*info.Sectors)

	info.ActiveSectors = make(DiskSectors, 0)

	activeData := make([]byte, 0)

	var err error
	info.Bitmap, err = dsk.PascalUsedBitmap()
	if err != nil {
		l.Errorf("Error reading bitmap: %s", err.Error())
		return
	}

	for b := 0; b < info.Blocks; b++ {

		if info.Bitmap[b] {

			data, _ := dsk.PRODOSGetBlock(b)

			t, s1, s2 := dsk.PRODOSGetBlockSectors(b)

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

			data, _ := dsk.PRODOSGetBlock(b)

			t, s1, s2 := dsk.PRODOSGetBlockSectors(b)

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

	// Analyzing files
	l.Log("Starting Analysis of files")

	files, err := dsk.PascalGetCatalog("*")
	if err != nil {
		l.Errorf("Problem reading directory: %s", err.Error())
		return
	}

	info.Files = make([]*DiskFile, 0)
	for _, fd := range files {
		l.Logf("- Name=%s, Type=%d, Len=%d", fd.GetName(), fd.GetType(), fd.GetFileSize())

		file := DiskFile{
			Filename: fd.GetName(),
			Type:     fd.GetType().String(),
			Locked:   fd.IsLocked(),
			Ext:      fd.GetType().Ext(),
			Created:  time.Now(),
			Modified: time.Now(),
		}

		//l.Log("start read")
		data, err := dsk.PascalReadFile(fd)
		if err == nil {
			sum := sha256.Sum256(data)
			file.SHA256 = hex.EncodeToString(sum[:])
			file.Size = len(data)
			if *ingestMode&1 == 1 {
				// text ingestion
				if fd.GetType() == disk.FileType_PAS_TEXT {
					file.Text = disk.StripText(data)
					file.Data = data
					file.TypeCode = TypeMask_Pascal | TypeCode(fd.GetType())
				} else {
					file.Data = data
					file.TypeCode = TypeMask_Pascal | TypeCode(fd.GetType())
				}
			}
		}

		//l.Log("end read")

		info.Files = append(info.Files, &file)

	}

	exists := exists(*baseName + "/" + info.GetFilename())

	if !exists || *forceIngest {
		info.WriteToFile(*baseName + "/" + info.GetFilename())
	} else {
		l.Log("Not writing as it already exists")
	}

	out(dsk.Format)

}
