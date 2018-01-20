package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
)

func inList(item string, list []string) bool {
	for _, v := range list {
		if strings.ToLower(v) == strings.ToLower(item) {
			return true
		}
	}
	return false
}

const EXCLUDEZEROBYTE = true
const EXCLUDEHELLO = true

func GetAllFiles(pattern string, pathfilter []string) map[string]DiskCatalog {

	cache := make(map[string]DiskCatalog)

	exists, matches := existsPattern(*baseName, pathfilter, pattern)
	if !exists {
		return cache
	}

	workchan := make(chan string, 100)
	var s sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < ingestWorkers; i++ {
		wg.Add(1)
		go func() {
			for m := range workchan {
				item := &Disk{}
				if err := item.ReadFromFile(m); err == nil {

					if len(item.Files) == 0 {
						continue
					}

					// Load cache
					s.Lock()
					cache[item.FullPath] = item.Files
					s.Unlock()

				} else {
					fmt.Println("FAIL")
				}
			}
			wg.Done()
		}()
	}

	var lastPc int = -1
	for i, m := range matches {

		//fmt.Printf("Queue: %s\n", m)

		workchan <- m

		pc := int(100 * float64(i) / float64(len(matches)))

		if pc != lastPc {
			fmt.Print("\r")
			os.Stderr.WriteString(fmt.Sprintf("Caching data... %d%%   ", pc))
		}

		lastPc = pc
	}
	close(workchan)

	wg.Wait()

	return cache
}

type FileOverlapRecord struct {
	files   map[string]map[*DiskFile]*DiskFile
	percent map[string]float64
	missing map[string][]*DiskFile
	extras  map[string][]*DiskFile
}

func (f *FileOverlapRecord) Remove(key string) {
	delete(f.files, key)
	delete(f.percent, key)
	delete(f.missing, key)
	delete(f.extras, key)
}

func (f *FileOverlapRecord) IsSubsetOf(filename string) bool {

	// f is a subset if:
	// missing == 0
	// extra > 0

	if _, ok := f.files[filename]; !ok {
		return false
	}

	return len(f.extras[filename]) > 0 && len(f.missing[filename]) == 0

}

func (f *FileOverlapRecord) IsSupersetOf(filename string) bool {

	// f is a superset if:
	// missing > 0
	// extra == 0

	if _, ok := f.files[filename]; !ok {
		return false
	}

	return len(f.extras[filename]) == 0 && len(f.missing[filename]) > 0

}

// Actual fuzzy file match report
func CollectFilesOverlapsAboveThreshold(t float64, pathfilter []string) map[string]*FileOverlapRecord {

	filerecords := GetAllFiles("*_*_*_*.fgp", pathfilter)

	results := make(map[string]*FileOverlapRecord)

	workchan := make(chan string, 100)
	var wg sync.WaitGroup
	var s sync.Mutex

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for i := 0; i < processWorkers; i++ {
		wg.Add(1)
		go func() {
			for m := range workchan {

				v := &FileOverlapRecord{
					files:   make(map[string]map[*DiskFile]*DiskFile),
					percent: make(map[string]float64),
					missing: make(map[string][]*DiskFile),
					extras:  make(map[string][]*DiskFile),
				}

				d := filerecords[m]

				for k, b := range filerecords {
					if k == m {
						continue // dont compare ourselves
					}
					// ok good to compare -- only keep if we need our threshold

					if closeness := CompareCatalogs(d, b, v, k); closeness < t {
						v.Remove(k)
					} else {
						v.percent[k] = closeness
					}
				}

				// since we delete < threshold, only add if we have any result
				if len(v.percent) > 0 {
					//os.Stderr.WriteString("\r\nAdded file: " + m + "\r\n\r\n")
					s.Lock()
					results[m] = v
					s.Unlock()
				}

			}
			wg.Done()
		}()
	}

	// feed data in
	var lastPc int = -1
	var i int
	for k, _ := range filerecords {

		if len(c) > 0 {
			sig := <-c
			if sig == os.Interrupt {
				close(c)
				os.Stderr.WriteString("\r\nInterrupted. Waiting for workers to stop.\r\n\r\n")
				break
			}
		}

		workchan <- k

		pc := int(100 * float64(i) / float64(len(filerecords)))

		if pc != lastPc {
			fmt.Print("\r")
			os.Stderr.WriteString(fmt.Sprintf("Processing files data... %d%%   ", pc))
		}

		lastPc = pc
		i++
	}

	close(workchan)
	wg.Wait()

	return results

}

func GetCatalogMap(d DiskCatalog) map[string]*DiskFile {

	out := make(map[string]*DiskFile)
	for _, v := range d {
		out[v.SHA256] = v
	}
	return out

}

func CompareCatalogs(d, b DiskCatalog, r *FileOverlapRecord, key string) float64 {

	var sameFiles float64
	var missingFiles float64
	var extraFiles float64

	var dmap = GetCatalogMap(d)
	var bmap = GetCatalogMap(b)

	for fileCk, info := range dmap {

		if info.Size == 0 && EXCLUDEZEROBYTE {
			continue
		}

		if info.Filename == "hello" && EXCLUDEHELLO {
			continue
		}

		binfo, bEx := bmap[fileCk]

		if bEx {
			sameFiles += 1
			// file match
			if r.files[key] == nil {
				r.files[key] = make(map[*DiskFile]*DiskFile)
			}
			//fmt.Printf("*** %s: %s -> %s\n", b.Filename, binfo.Filename, info.Filename)
			r.files[key][binfo] = info
		} else {
			missingFiles += 1
			// file match
			if r.missing[key] == nil {
				r.missing[key] = make([]*DiskFile, 0)
			}
			//fmt.Printf("*** %s: %s -> %s\n", b.Filename, binfo.Filename, info.Filename)
			r.missing[key] = append(r.missing[key], info)
		}

	}

	for fileCk, info := range bmap {

		if info.Size == 0 {
			continue
		}

		_, dEx := dmap[fileCk]

		if !dEx {
			extraFiles += 1
			// file match
			if r.extras[key] == nil {
				r.extras[key] = make([]*DiskFile, 0)
			}
			//fmt.Printf("*** %s: %s -> %s\n", b.Filename, binfo.Filename, info.Filename)
			r.extras[key] = append(r.extras[key], info)
		}

	}

	if (sameFiles + extraFiles + missingFiles) == 0 {
		return 0
	}

	// return sameSectors / dTotal, sameSectors / bTotal, diffSectors / dTotal, diffSectors / btotal
	return sameFiles / (sameFiles + extraFiles + missingFiles)

}

func CollectFileSubsets(pathfilter []string) map[string]*FileOverlapRecord {

	filerecords := GetAllFiles("*_*_*_*.fgp", pathfilter)

	results := make(map[string]*FileOverlapRecord)

	workchan := make(chan string, 100)
	var wg sync.WaitGroup
	var s sync.Mutex

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for i := 0; i < processWorkers; i++ {
		wg.Add(1)
		go func() {
			for m := range workchan {

				v := &FileOverlapRecord{
					files:   make(map[string]map[*DiskFile]*DiskFile),
					percent: make(map[string]float64),
					missing: make(map[string][]*DiskFile),
					extras:  make(map[string][]*DiskFile),
				}

				d := filerecords[m]

				for k, b := range filerecords {
					if k == m {
						continue // dont compare ourselves
					}
					// ok good to compare -- only keep if we need our threshold

					closeness := CompareCatalogs(d, b, v, k)
					if !v.IsSubsetOf(k) {
						v.Remove(k)
					} else {
						v.percent[k] = closeness
					}
				}

				// since we delete < threshold, only add if we have any result
				if len(v.percent) > 0 {
					//os.Stderr.WriteString("\r\nAdded file: " + m + "\r\n\r\n")
					s.Lock()
					results[m] = v
					s.Unlock()
				}

			}
			wg.Done()
		}()
	}

	// feed data in
	var lastPc int = -1
	var i int
	for k, _ := range filerecords {

		if len(c) > 0 {
			sig := <-c
			if sig == os.Interrupt {
				close(c)
				os.Stderr.WriteString("\r\nInterrupted. Waiting for workers to stop.\r\n\r\n")
				break
			}
		}

		workchan <- k

		pc := int(100 * float64(i) / float64(len(filerecords)))

		if pc != lastPc {
			fmt.Print("\r")
			os.Stderr.WriteString(fmt.Sprintf("Processing files data... %d%%   ", pc))
		}

		lastPc = pc
		i++
	}

	close(workchan)
	wg.Wait()

	return results

}

func CollectFilesOverlapsCustom(keep func(d1, d2 string, v *FileOverlapRecord) bool, pathfilter []string) map[string]*FileOverlapRecord {

	filerecords := GetAllFiles("*_*_*_*.fgp", pathfilter)

	results := make(map[string]*FileOverlapRecord)

	workchan := make(chan string, 100)
	var wg sync.WaitGroup
	var s sync.Mutex

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for i := 0; i < processWorkers; i++ {
		wg.Add(1)
		go func() {
			for m := range workchan {

				v := &FileOverlapRecord{
					files:   make(map[string]map[*DiskFile]*DiskFile),
					percent: make(map[string]float64),
					missing: make(map[string][]*DiskFile),
					extras:  make(map[string][]*DiskFile),
				}

				d := filerecords[m]

				for k, b := range filerecords {
					if k == m {
						continue // dont compare ourselves
					}
					// ok good to compare -- only keep if we need our threshold
					closeness := CompareCatalogs(d, b, v, k)

					if !keep(m, k, v) {
						v.Remove(k)
					} else {
						v.percent[k] = closeness
					}
				}

				// since we delete < threshold, only add if we have any result
				if len(v.files) > 0 {
					//os.Stderr.WriteString("\r\nAdded file: " + m + "\r\n\r\n")
					s.Lock()
					results[m] = v
					s.Unlock()
				}

			}
			wg.Done()
		}()
	}

	// feed data in
	var lastPc int = -1
	var i int
	for k, _ := range filerecords {

		if len(c) > 0 {
			sig := <-c
			if sig == os.Interrupt {
				close(c)
				os.Stderr.WriteString("\r\nInterrupted. Waiting for workers to stop.\r\n\r\n")
				break
			}
		}

		workchan <- k

		pc := int(100 * float64(i) / float64(len(filerecords)))

		if pc != lastPc {
			fmt.Print("\r")
			os.Stderr.WriteString(fmt.Sprintf("Processing files data... %d%%   ", pc))
		}

		lastPc = pc
		i++
	}

	close(workchan)
	wg.Wait()

	return results

}
