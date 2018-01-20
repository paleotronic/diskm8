package main

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/paleotronic/diskm8/disk"
	"github.com/paleotronic/diskm8/loggy"
)

func analyzeRDOS(id int, dsk *disk.DSKWrapper, info *Disk) {

	l := loggy.Get(id)

	// Sector bitmap
	l.Logf("Reading Disk Structure...")

	info.Tracks, info.Sectors = 35, dsk.RDOSFormat.Spec().SectorMax

	l.Logf("Tracks: %d, Sectors: %d", info.Tracks, info.Sectors)

	l.Logf("Reading sector bitmap and SHA256'ing sectors")

	info.Bitmap = make([]bool, info.Tracks*info.Sectors)

	info.ActiveSectors = make(DiskSectors, 0)
	info.InactiveSectors = make(DiskSectors, 0)

	activeData := make([]byte, 0)

	var err error
	info.Bitmap, err = dsk.RDOSUsedBitmap()
	if err != nil {
		l.Errorf("Error reading bitmap: %s", err.Error())
		return
	}

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

	files, err := dsk.RDOSGetCatalog("*")
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
			Ext:      fd.Type().Ext(),
		}

		//l.Log("start read")
		data, err := dsk.RDOSReadFile(fd)
		if err == nil {
			sum := sha256.Sum256(data)
			file.SHA256 = hex.EncodeToString(sum[:])
			file.Size = len(data)
			if *ingestMode&1 == 1 {
				if fd.Type() == disk.FileType_RDOS_AppleSoft {
					file.Text = disk.ApplesoftDetoks(data)
					file.TypeCode = TypeMask_RDOS | TypeCode(fd.Type())
					file.Data = data
				} else if fd.Type() == disk.FileType_RDOS_Text {
					file.Text = disk.StripText(data)
					file.TypeCode = TypeMask_RDOS | TypeCode(fd.Type())
					file.Data = data
				} else {
					file.Data = data
					file.LoadAddress = fd.LoadAddress()
					file.TypeCode = TypeMask_RDOS | TypeCode(fd.Type())
				}
			}
		}
		//l.Log("end read")

		l.Logf("FILETEXT=\n%s", dump(data))

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
