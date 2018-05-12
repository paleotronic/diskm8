package main

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/paleotronic/diskm8/disk"
	"github.com/paleotronic/diskm8/loggy"
)

func analyzeDOS13(id int, dsk *disk.DSKWrapper, info *Disk) {

	l := loggy.Get(id)

	// Sector bitmap
	l.Logf("Reading Disk VTOC...")
	vtoc, err := dsk.AppleDOSGetVTOC()
	if err != nil {
		l.Errorf("Error reading VTOC: %s", err.Error())
		return
	}

	info.Tracks, info.Sectors = vtoc.GetTracks(), vtoc.GetSectors()
	l.Logf("Tracks: %d, Sectors: %d", info.Tracks, info.Sectors)

	l.Logf("Reading sector bitmap and SHA256'ing sectors")

	info.Bitmap = make([]bool, info.Tracks*info.Sectors)

	info.ActiveSectors = make(DiskSectors, 0)
	info.InactiveSectors = make(DiskSectors, 0)

	activeData := make([]byte, 0)

	for t := 0; t < info.Tracks; t++ {

		for s := 0; s < info.Sectors; s++ {
			info.Bitmap[t*info.Sectors+s] = !vtoc.IsTSFree(t, s)

			// checksum sector
			//info.SectorFingerprints[dsk.ChecksumSector(t, s)] = &DiskBlock{Track: t, Sector: s}

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
			} else {
				sector := &DiskSector{
					Track:  t,
					Sector: s,
					SHA256: dsk.ChecksumSector(t, s),
				}

				data := dsk.Read()
				if *ingestMode&2 == 2 {
					sector.Data = data
				}
				//activeData = append(activeData, data...)

				info.InactiveSectors = append(info.InactiveSectors, sector)
			}
		}

	}

	sum := sha256.Sum256(activeData)
	info.SHA256Active = hex.EncodeToString(sum[:])

	info.LogBitmap(id)

	// Analyzing files
	l.Log("Starting Analysis of files")

	vtoc, files, err := dsk.AppleDOSGetCatalog("*")
	if err != nil {
		l.Errorf("Problem reading directory: %s", err.Error())
		return
	}

	info.Files = make([]*DiskFile, 0)
	for _, fd := range files {
		l.Logf("- Name=%s, Type=%s", fd.NameUnadorned(), fd.Type())

		file := DiskFile{
			Filename: fd.NameUnadorned(),
			Type:     fd.Type().String(),
			Locked:   fd.IsLocked(),
			Ext:      fd.Type().Ext(),
			Created:  time.Now(),
			Modified: time.Now(),
		}

		size, addr, data, err := dsk.AppleDOSReadFileRaw(fd)
		if err == nil {
			sum := sha256.Sum256(data)
			file.SHA256 = hex.EncodeToString(sum[:])
			file.Size = size
			if *ingestMode&1 == 1 {
				if fd.Type() == disk.FileTypeAPP {
					file.Text = disk.ApplesoftDetoks(data)
					file.TypeCode = TypeMask_AppleDOS | TypeCode(fd.Type())
					file.Data = data
					file.LoadAddress = 0x801
				} else if fd.Type() == disk.FileTypeINT {
					file.Text = disk.IntegerDetoks(data)
					file.Data = data
					file.TypeCode = TypeMask_AppleDOS | TypeCode(fd.Type())
					file.LoadAddress = 0x1000
				} else if fd.Type() == disk.FileTypeTXT {
					file.Text = disk.StripText(data)
					file.Data = data
					file.TypeCode = TypeMask_AppleDOS | TypeCode(fd.Type())
					file.LoadAddress = 0x0000
				} else if fd.Type() == disk.FileTypeBIN && len(data) >= 2 {
					file.LoadAddress = addr
					file.Data = data
					file.TypeCode = TypeMask_AppleDOS | TypeCode(fd.Type())
				} else {
					file.LoadAddress = 0x0000
					file.Data = data
					file.TypeCode = TypeMask_AppleDOS | TypeCode(fd.Type())
				}
			}
		}

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
