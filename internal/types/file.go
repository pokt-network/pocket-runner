package types

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
)

// numberedBackupFile matches files that looks like file.ext.~1~ and uses a capture group to grab the number
var numberedBackupFile = regexp.MustCompile(`^.*\.~([0-9]{1,5})~$`)

var (
	// ErrFileNotExist occurs when a file is given that does not exist when its existence is required.
	ErrFileNotExist = errors.New("no such file or directory")
	// ErrCannotOpenSrc occurs when a src file cannot be opened with os.Open().
	ErrCannotOpenSrc = errors.New("source file cannot be opened")
	// ErrCannotStatFile occurs when a file receives an error from get os.Stat().
	ErrCannotStatFile = errors.New("cannot stat file, check that file path is accessible")
	// ErrCannotChmodFile occurs when an error is received trying to change permissions on a file.
	ErrCannotChmodFile = errors.New("cannot change permissions on file")
	// ErrCannotCreateTmpFile occurs when an error is received attempting to create a temporary file for atomic copy.
	ErrCannotCreateTmpFile = errors.New("temp file cannot be created")
	// ErrCannotOpenOrCreateDstFile occurs when an error is received attempting to open or create destination file during non-atomic copy.
	ErrCannotOpenOrCreateDstFile = errors.New("destination file cannot be created")
	// ErrCannotRenameTempFile occurs when an error is received trying to rename the temporary copy file to the destination.
	ErrCannotRenameTempFile = errors.New("cannot rename temp file, check file or directory permissions")
	// ErrOmittingDir occurs when attempting to copy a directory but Options.Recursive is not set to true.
	ErrOmittingDir = errors.New("Options.Recursive is not true, omitting directory")
	// ErrWithParentsDstMustBeDir occurs when the destination is expected to be an existing directory but is not
	// present or accessible.
	ErrWithParentsDstMustBeDir = errors.New("with Options.Parents, the destination must be a directory")
	// ErrCannotOverwriteNonDir occurs when attempting to copy a directory to a non-directory.
	ErrCannotOverwriteNonDir = errors.New("cannot overwrite non-directory")
	// ErrReadingSrcDir occurs when attempting to read contents of the source directory fails
	ErrReadingSrcDir = errors.New("cannot read source directory, check source directory permissions")
	// ErrWritingFileToExistingDir occurs when attempting to write a file to an existing directory.
	// See AppendNameToPath option for a more dynamic approach.
	ErrWritingFileToExistingDir = errors.New("cannot overwrite existing directory with file")
	// ErrInvalidBackupControlValue occurs when a control value is given to the Backup option, but the value is invalid.
	ErrInvalidBackupControlValue = errors.New("invalid backup value, valid values are 'off', 'simple', 'existing', 'numbered'")
)

// File describes a file and associated options for operations on the file.
type File struct {
	// Path is the path to the src file.
	Path string
	// fileInfoOnInit is os.FileInfo for file when initialized.
	fileInfoOnInit os.FileInfo
	// existOnInit is true if the file exists when initialized.
	existOnInit bool
	// isDir is true if the file object is a directory.
	isDir bool
}

// NewFile creates a new File.
func NewFile(path string) *File {
	return &File{Path: path}
}

// setInfo will collect information about a File and populate the necessary fields.
func (f *File) setInfo() error {
	info, err := os.Lstat(f.Path)
	f.fileInfoOnInit = info
	if err != nil {
		if !os.IsNotExist(err) {
			// if we are here then we have an error, but not one indicating the file does not exist
			return err
		}
	} else {
		f.existOnInit = true
		if f.fileInfoOnInit.IsDir() {
			f.isDir = true
		}
	}
	return nil
}

func (f *File) isSymlink() bool {
	if f.fileInfoOnInit.Mode()&os.ModeSymlink != 0 {
		return true
	}
	return false
}

// shouldMakeParents returns true if we should make parent directories up to the dst
func (f *File) shouldMakeParents(opts Options) bool {
	if opts.MkdirAll || opts.mkdirAll {
		return true
	}

	if opts.Parents {
		return true
	}

	if f.existOnInit {
		return false
	}

	parent := filepath.Dir(filepath.Clean(f.Path))
	if _, err := os.Stat(parent); !os.IsNotExist(err) {
		// dst does not exist but the direct parent does. make the target dir.
		return true
	}

	return false
}

// shouldCopyParents returns true if parent directories from src should be copied into dst.
func (f *File) shouldCopyParents(opts Options) bool {
	if !opts.Parents {
		return false
	}
	return true
}
