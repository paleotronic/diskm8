package disk

import "time"

type CatalogEntryType int

const (
	CETUnknown CatalogEntryType = iota
	CETBinary
	CETBasicApplesoft
	CETBasicInteger
	CETPascal
	CETText
	CETData
	CETGraphics
)

type CatalogEntry interface {
	Size() int // file size in bytes
	Name() string
	NameUnadorned() string
	Date() time.Time
	Type() CatalogEntryType
}

type DiskImage interface {
	IsValid() (bool, DiskFormat, SectorOrder)
	GetCatalog(path string, pattern string) ([]CatalogEntry, error)
	ReadFile(fd CatalogEntry) (int, []byte, error)
	StoreFile(fd CatalogEntry) error
	GetUsedBitmap() ([]bool, error)
	Nibblize() ([]byte, error)
}
