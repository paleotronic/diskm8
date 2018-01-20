package main

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/paleotronic/diskm8/disk"
	"github.com/paleotronic/diskm8/loggy"
)

func analyzePRODOS16(id int, dsk *disk.DSKWrapper, info *Disk) {

	l := loggy.Get(id)

	isPD, Format, Layout := dsk.IsProDOS()
	l.Logf("IsProDOS=%v, Format=%s, Layout=%d", isPD, Format, Layout)
	if isPD {
		dsk.Layout = Layout
	}

	// Sector bitmap
	l.Logf("Reading Disk VTOC...")
	vtoc, err := dsk.PRODOSGetVDH(2)
	if err != nil {
		l.Errorf("Error reading VTOC: %s", err.Error())
		return
	}

	info.Blocks = vtoc.GetTotalBlocks()

	l.Logf("Filecount: %d", vtoc.GetFileCount())

	l.Logf("Blocks: %d", info.Blocks)

	l.Logf("Reading sector bitmap and SHA256'ing sectors")

	info.Bitmap = make([]bool, info.Blocks)

	info.ActiveSectors = make(DiskSectors, 0)

	activeData := make([]byte, 0)

	vbitmap, err := dsk.PRODOSGetVolumeBitmap()
	if err != nil {
		l.Errorf("Error reading volume bitmap: %s", err.Error())
		return
	}

	l.Debug(vbitmap)

	for b := 0; b < info.Blocks; b++ {
		info.Bitmap[b] = !vbitmap.IsBlockFree(b)

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

	// // Analyzing files
	l.Log("Starting Analysis of files")

	prodosDir(id, 2, "", dsk, info)

	exists := exists(*baseName + "/" + info.GetFilename())

	if !exists || *forceIngest {
		e := info.WriteToFile(*baseName + "/" + info.GetFilename())
		if e != nil {
			l.Errorf("Error writing fingerprint: %v", e)
			panic(e)
		}
	} else {
		l.Log("Not writing as it already exists")
	}

	out(dsk.Format)

}

func prodosDir(id int, start int, path string, dsk *disk.DSKWrapper, info *Disk) {

	l := loggy.Get(id)

	_, files, err := dsk.PRODOSGetCatalog(start, "*")
	if err != nil {
		l.Errorf("Problem reading directory: %s", err.Error())
		return
	}
	if info.Files == nil {
		info.Files = make([]*DiskFile, 0)
	}
	for _, fd := range files {
		l.Logf("- Path=%s, Name=%s, Type=%s", path, fd.NameUnadorned(), fd.Type())

		var file DiskFile

		if path == "" {

			file = DiskFile{
				Filename: fd.NameUnadorned(),
				Type:     fd.Type().String(),
				Locked:   fd.IsLocked(),
				Ext:      fd.Type().Ext(),
				Created:  fd.CreateTime(),
				Modified: fd.ModTime(),
			}

		} else {

			file = DiskFile{
				Filename: path + "/" + fd.NameUnadorned(),
				Type:     fd.Type().String(),
				Locked:   fd.IsLocked(),
				Ext:      fd.Type().Ext(),
				Created:  fd.CreateTime(),
				Modified: fd.ModTime(),
			}

		}

		if fd.Type() != disk.FileType_PD_Directory {
			_, _, data, err := dsk.PRODOSReadFileRaw(fd)
			if err == nil {
				sum := sha256.Sum256(data)
				file.SHA256 = hex.EncodeToString(sum[:])
				file.Size = len(data)
				if *ingestMode&1 == 1 {
					if fd.Type() == disk.FileType_PD_APP {
						file.Text = disk.ApplesoftDetoks(data)
						file.TypeCode = TypeMask_ProDOS | TypeCode(fd.Type())
						file.Data = data
						file.LoadAddress = fd.AuxType()
					} else if fd.Type() == disk.FileType_PD_INT {
						file.Text = disk.IntegerDetoks(data)
						file.TypeCode = TypeMask_ProDOS | TypeCode(fd.Type())
						file.Data = data
						file.LoadAddress = fd.AuxType()
					} else if fd.Type() == disk.FileType_PD_TXT {
						file.Text = disk.StripText(data)
						file.Data = data
						file.TypeCode = TypeMask_ProDOS | TypeCode(fd.Type())
						file.LoadAddress = fd.AuxType()
					} else {
						file.LoadAddress = fd.AuxType()
						file.Data = data
						file.TypeCode = TypeMask_ProDOS | TypeCode(fd.Type())
					}
				}
			}
		}

		info.Files = append(info.Files, &file)

		if fd.Type() == disk.FileType_PD_Directory {
			newpath := path
			if path != "" {
				newpath += "/" + fd.NameUnadorned()
			} else {
				newpath = fd.NameUnadorned()
			}
			prodosDir(id, fd.IndexBlock(), newpath, dsk, info)
		}

	}

}
