## Usage examples

Ingest your disk collection, so diskm8 can report on them:

```
diskm8 -ingest C:\Users\myname\LotsOfDisks
```

Find Whole Disk duplicates:

```
diskm8 -whole-dupes 
```

Find Active Sectors duplicates (inactive sectors can be different):

```
diskm8 -as-dupes
```

Find Duplicate files across disks:

```
diskm8 -file-dupes
```

Find Active Sector duplicates but only under a folder:

```
diskm8 -as-dupes -select "C:\Users\myname\LotsOfDisks\Operating Systems"
```

