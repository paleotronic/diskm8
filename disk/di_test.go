package disk

import (
	//"strings"
	"testing"
)
import "fmt"

//import "os"

func TestDisk(t *testing.T) {

	if STD_DISK_BYTES != 143360 {
		t.Error(fmt.Sprintf("Wrong size got %d", STD_DISK_BYTES))
	}

	dsk, e := NewDSKWrapper("g19.dsk")
	if e != nil {
		t.Error(e)
	}

	fmt.Printf("Disk format is %d\n", dsk.Format)

	_, fdlist, e := dsk.GetCatalogProDOSPathed(2, "GAMES", "")
	for _, fd := range fdlist {
		fmt.Printf("[%s]\n", fd.Name())
	}

	t.Fail()

}
