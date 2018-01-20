## Usage examples

Ingest your disk collection, so dskalyzer can report on them:

```
dskalyzer -ingest C:\Users\myname\LotsOfDisks
```

Find Whole Disk duplicates:

```
dskalyzer -whole-dupes 
```

Find Active Sectors duplicates (inactive sectors can be different):

```
dskalyzer -as-dupes
```

Find Duplicate files across disks:

```
dskalyzer -file-dupes
```

Find Active Sector duplicates but only under a folder:

```
dskalyzer -as-dupes -select "C:\Users\myname\LotsOfDisks\Operating Systems"
```

