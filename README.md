DSKalyzer is a cross-platform command-line tool for manipulating and managing Apple II DSK (and other) images. 

Download from: https://github.com/paleotronic/dskalyzer/releases

Features include:

- Read from ProDOS, DOS 3.X, RDOS and Pascal disk images; 
- ProDOS or DOS ordered; 2MG and NIB; 113-800K
- Write to Prodos and DOS 3.3 disk images;
- Extract and convert binary, text and detokenize BASIC files (Integer and Applesoft);
- Write binary, text and retokenized BASIC (Applesoft) files back to disk images;
- Copy and move files between disk images; delete files, create new folders (ProDOS), etc;
- Generate disk reports that provide track and sector information, text extraction and more;
- Compare multiple disks to determine duplication, or search disks for text or filenames.
- Use command-line flags (allows for automation) or an interactive shell;
- Builds for MacOS, Windows, Linux, FreeBSD and Raspberry Pi.
- Open source; GPLv3 licensed.
- Written in Go!

DSKalyzer is a command line tool for analyzing and managing Apple II DSK images and their archives. Its features include not only the standard set of disk manipulation tools -- extract (with text conversion), import to disk (including tokenisation of Applesoft BASIC), delete, and so forth -- but also the ability to identify duplicates — complete, active sector, and subset; find file, sector and other commonalities between disks (including as a percentage of similarity or difference); search de-tokenized BASIC, text and binary / sector data; generate reports identifying and / or collating disk type, DOS, geometry, size, and so forth; allowing for easier, semi-automated DSK archival management and research.

DSKalyzer works by first “ingesting” your disk(s), creating an index containing various pieces of information (disk / sector / file hashes, catalogs, text data, binary data etc.) about each disk that is then searchable using the same tool. This way you can easily find information stored on disks without tediously searching manually or through time-consuming multiple image scans. You can also identify duplicates, quasi-duplicates (disks with only minor differences or extraneous data), or iterations, reducing redundancies.

Once you've identified a search you can also extract selected files. DSKalyzer can report to standard output (terminal), to a text file, or to a CSV file.
```
Shell commands (executing DSKalyzer without flags enters shell):

Note: You must mount a disk before performing tasks on it.

analyze    Process disk using dskalyzer analytics
cat        Display file information
cd         Change local path
copy       Copy files from one volume to another
delete     Remove file from disk
disks      List mounted volumes
extract    extract file from disk image
help       Shows this help
info       Information about the current disk
ingest     Ingest directory containing disks (or single disk) into system
lock       Lock file on the disk
ls         List local files
mkdir      Create a directory on disk
mount      Mount a disk image
move       Move files from one volume to another
put        Copy local file to disk
quarantine Like report, but allow moving dupes to a backup folder
quit       Leave this place
rename     Rename a file on the disk
report     Run a report
target     Select mounted volume as default
unlock     Unlock file on the disk
unmount    unmount disk image

Command-line flags: 

(Note: You must ingest your disk library before you can run comparison or search operations on it)

  -active-sector-partial
    	Run partial sector match (active only) against all disks
  -active-sector-subset
    	Run subset (active) sector match against all disks
  -adorned
    	Extract files named similar to CP (default true)
  -all-file-partial
    	Run partial file match against all disks
  -all-file-subset
    	Run subset file match against all disks
  -all-sector-partial
    	Run partial sector match (all) against all disks
  -all-sector-subset
    	Run subset (non-zero) sector match against all disks
  -as-dupes
    	Run active sectors only disk dupe report
  -as-partial
    	Run partial active sector match against single disk (-disk required)
  -c	Cache data to memory for quicker processing (default true)
  -cat-dupes
    	Run duplicate catalog report
  -catalog
    	List disk contents (-with-disk)
  -csv
    	Output data to CSV format
  -datastore string
    	Database of disk fingerprints for checking (default "/Users/melody/DSKalyzer/fingerprints")
  -dir
    	Directory specified disk (needs -disk)
  -dir-create string
    	Directory to create (-with-disk)
  -dir-format string
    	Format of dir (default "{filename} {type} {size:kb} Checksum: {sha256}")
  -extract string
    	Extract files/disks matched in searches ('#'=extract disk, '@'=extract files)
  -file string
    	Search for other disks containing file
  -file-delete string
    	File to delete (-with-disk)
  -file-dupes
    	Run file dupe report
  -file-extract string
    	File to delete from disk (-with-disk)
  -file-partial
    	Run partial file match against single disk (-disk required)
  -file-put string
    	File to put on disk (-with-disk)
  -force
    	Force re-ingest disks that already exist
  -ingest string
    	Disk file or path to ingest
  -ingest-mode int
    	Ingest mode:
	0=Fingerprints only
	1=Fingerprints + text
	2=Fingerprints + sector data
	3=All (default 1)
  -max-diff int
    	Maximum different # files for -all-file-partial
  -min-same int
    	Minimum same # files for -all-file-partial
  -out string
    	Output file (empty for stdout)
  -quarantine
    	Run -as-dupes and -whole-disk in quarantine mode
  -query string
    	Disk file to query or analyze
  -search-filename string
    	Search database for file with name
  -search-sha string
    	Search database for file with checksum
  -search-text string
    	Search database for file containing text
  -select
    	Select files for analysis or search based on file/dir/mask
  -shell
    	Start interactive mode
  -shell-batch string
    	Execute shell command(s) from file and exit
  -similarity float
    	Object match threshold for -*-partial reports (default 0.9)
  -verbose
    	Log to stderr
  -whole-dupes
    	Run whole disk dupe report
  -with-disk string
    	Perform disk operation (-file-extract,-file-put,-file-delete)
```
Getting Started

Ingest your disk collection, so dskalyzer can report on them:

dskalyzer -ingest "C:\Users\myname\LotsOfDisks"
Simple Reports

Find Whole Disk duplicates:

dskalyzer -whole-dupes 
Find Active Sectors duplicates (inactive sectors can be different):

dskalyzer -as-dupes
Find Duplicate files across disks:

dskalyzer -file-dupes
Limiting reports to subdirectories

Find Active Sector duplicates but only under a folder:

dskalyzer -as-dupes -select "C:\Users\myname\LotsOfDisks\Operating Systems"
```
