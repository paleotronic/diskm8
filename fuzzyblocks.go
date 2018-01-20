package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
)

const EMPTYSECTOR = "5341e6b2646979a70e57653007a1f310169421ec9bdd9f1a5648f75ade005af1"

func GetAllDiskSectors(pattern string, pathfilter []string) map[string]DiskSectors {

	cache := make(map[string]DiskSectors)

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

					//
					chunk := append(item.ActiveSectors, item.InactiveSectors...)
					tmp := make(DiskSectors, 0)
					for _, v := range chunk {
						if v.SHA256 != EMPTYSECTOR {
							tmp = append(tmp, v)
						} else {
							fmt.Printf("%s: throw away zero sector T%d,S%d\n", item.Filename, v.Track, v.Sector)
						}
					}

					// Load cache
					s.Lock()
					cache[item.FullPath] = tmp
					s.Unlock()

				}
			}
			wg.Done()
		}()
	}

	var lastPc int = -1
	for i, m := range matches {

		workchan <- m

		pc := int(100 * float64(i) / float64(len(matches)))

		if pc != lastPc {
			fmt.Print("\r")
			os.Stderr.WriteString(fmt.Sprintf("Caching disk sector data... %d%%   ", pc))
		}

		lastPc = pc
	}
	close(workchan)

	wg.Wait()

	return cache
}

func GetActiveDiskSectors(pattern string, pathfilter []string) map[string]DiskSectors {

	cache := make(map[string]DiskSectors)

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

					// Load cache
					s.Lock()
					cache[item.FullPath] = item.ActiveSectors
					s.Unlock()

				}
			}
			wg.Done()
		}()
	}

	var lastPc int = -1
	for i, m := range matches {

		workchan <- m

		pc := int(100 * float64(i) / float64(len(matches)))

		if pc != lastPc {
			fmt.Print("\r")
			os.Stderr.WriteString(fmt.Sprintf("Caching disk sector data... %d%%   ", pc))
		}

		lastPc = pc
	}
	close(workchan)

	wg.Wait()

	return cache
}

func GetSectorMap(d DiskSectors) map[string]*DiskSector {

	out := make(map[string]*DiskSector)
	for _, v := range d {
		out[fmt.Sprintf("T%d,S%d", v.Track, v.Sector)] = v
	}
	return out

}

type SectorOverlapRecord struct {
	same    map[string]map[*DiskSector]*DiskSector
	percent map[string]float64
	missing map[string][]*DiskSector
	extras  map[string][]*DiskSector
}

func (f *SectorOverlapRecord) Remove(key string) {
	delete(f.same, key)
	delete(f.percent, key)
	delete(f.missing, key)
	delete(f.extras, key)
}

func (f *SectorOverlapRecord) IsSubsetOf(filename string) bool {

	// f is a subset if:
	// missing == 0
	// extra > 0

	if _, ok := f.same[filename]; !ok {
		return false
	}

	return len(f.extras[filename]) > 0 && len(f.missing[filename]) == 0

}

func (f *SectorOverlapRecord) IsSupersetOf(filename string) bool {

	// f is a superset if:
	// missing > 0
	// extra == 0

	if _, ok := f.same[filename]; !ok {
		return false
	}

	return len(f.extras[filename]) == 0 && len(f.missing[filename]) > 0

}

func CompareSectors(d, b DiskSectors, r *SectorOverlapRecord, key string) float64 {

	var sameSectors float64
	var missingSectors float64
	var extraSectors float64

	var dmap = GetSectorMap(d)
	var bmap = GetSectorMap(b)

	for fileCk, info := range dmap {

		binfo, bEx := bmap[fileCk]

		if bEx && info.SHA256 == binfo.SHA256 {
			sameSectors += 1
			if r.same[key] == nil {
				r.same[key] = make(map[*DiskSector]*DiskSector)
			}

			r.same[key][binfo] = info
		} else {
			missingSectors += 1
			if r.missing[key] == nil {
				r.missing[key] = make([]*DiskSector, 0)
			}
			r.missing[key] = append(r.missing[key], info)
		}

	}

	for fileCk, info := range bmap {

		_, dEx := dmap[fileCk]

		if !dEx {
			extraSectors += 1
			// file match
			if r.extras[key] == nil {
				r.extras[key] = make([]*DiskSector, 0)
			}
			//fmt.Printf("*** %s: %s -> %s\n", b.Filename, binfo.Filename, info.Filename)
			r.extras[key] = append(r.extras[key], info)
		}

	}

	if (sameSectors + extraSectors + missingSectors) == 0 {
		return 0
	}

	// return sameSectors / dTotal, sameSectors / bTotal, diffSectors / dTotal, diffSectors / btotal
	return sameSectors / (sameSectors + extraSectors + missingSectors)

}

// Actual fuzzy file match report
func CollectSectorOverlapsAboveThreshold(t float64, pathfilter []string, ff func(pattern string, pathfilter []string) map[string]DiskSectors) map[string]*SectorOverlapRecord {

	filerecords := ff("*_*_*_*.fgp", pathfilter)

	results := make(map[string]*SectorOverlapRecord)

	workchan := make(chan string, 100)
	var wg sync.WaitGroup
	var s sync.Mutex

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for i := 0; i < processWorkers; i++ {
		wg.Add(1)
		go func() {
			for m := range workchan {

				v := &SectorOverlapRecord{
					same:    make(map[string]map[*DiskSector]*DiskSector),
					percent: make(map[string]float64),
					missing: make(map[string][]*DiskSector),
					extras:  make(map[string][]*DiskSector),
				}

				d := filerecords[m]

				for k, b := range filerecords {
					if k == m {
						continue // dont compare ourselves
					}
					// ok good to compare -- only keep if we need our threshold

					if closeness := CompareSectors(d, b, v, k); closeness < t {
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
			os.Stderr.WriteString(fmt.Sprintf("Processing sectors data... %d%%   ", pc))
		}

		lastPc = pc
		i++
	}

	close(workchan)
	wg.Wait()

	return results

}

// Actual fuzzy file match report
func CollectSectorSubsets(pathfilter []string, ff func(pattern string, pathfilter []string) map[string]DiskSectors) map[string]*SectorOverlapRecord {

	filerecords := ff("*_*_*_*.fgp", pathfilter)

	results := make(map[string]*SectorOverlapRecord)

	workchan := make(chan string, 100)
	var wg sync.WaitGroup
	var s sync.Mutex

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for i := 0; i < processWorkers; i++ {
		wg.Add(1)
		go func() {
			for m := range workchan {

				v := &SectorOverlapRecord{
					same:    make(map[string]map[*DiskSector]*DiskSector),
					percent: make(map[string]float64),
					missing: make(map[string][]*DiskSector),
					extras:  make(map[string][]*DiskSector),
				}

				d := filerecords[m]

				for k, b := range filerecords {
					if k == m {
						continue // dont compare ourselves
					}
					// ok good to compare -- only keep if we need our threshold

					closeness := CompareSectors(d, b, v, k)

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
			os.Stderr.WriteString(fmt.Sprintf("Processing sectors data... %d%%   ", pc))
		}

		lastPc = pc
		i++
	}

	close(workchan)
	wg.Wait()

	return results

}
