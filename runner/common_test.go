package runner

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

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

// SimpleCopy will src to dst with default Options.
func SimpleCopy(src, dst string) error {
	return Copy(src, dst, Options{})
}

// Copy will copy src to dst.  Behavior is determined by the given Options.
func Copy(src, dst string, opts Options) (err error) {
	opts.setLoggers()
	srcFile, dstFile := NewFile(src), NewFile(dst)

	// set src attributes
	if err := srcFile.setInfo(); err != nil {
		return errors.Wrapf(ErrCannotStatFile, "source file %s: %s", srcFile.Path, err)
	}
	if !srcFile.existOnInit {
		return errors.Wrapf(ErrFileNotExist, "source file %s", srcFile.Path)
	}
	opts.logDebug("src %s existOnInit: %t", srcFile.Path, srcFile.existOnInit)

	// stat dst attributes. handle errors later
	_ = dstFile.setInfo()
	opts.logDebug("dst %s existOnInit: %t", dstFile.Path, dstFile.existOnInit)

	if dstFile.shouldMakeParents(opts) {
		opts.mkdirAll = true
		opts.DebugLogFunc("dst mkdirAll: true")
	}

	if opts.Parents {
		if dstFile.existOnInit && !dstFile.isDir {
			return ErrWithParentsDstMustBeDir
		}
		// TODO: figure out how to handle windows paths where they reference the full path like c:/dir
		dstFile.Path = filepath.Join(dstFile.Path, srcFile.Path)
		opts.logDebug("because of Parents option, setting dst Path to %s", dstFile.Path)
		dstFile.isDir = srcFile.isDir
		opts.Parents = false // ensure we don't keep creating parents on recursive calls
	}

	// copying src directory requires dst is also a directory, if it existOnInit
	if srcFile.isDir && dstFile.existOnInit && !dstFile.isDir {
		return errors.Wrapf(
			ErrCannotOverwriteNonDir, "source directory %s, destination file %s", srcFile.Path, dstFile.Path)
	}

	// divide and conquer
	switch {
	case opts.Link:
		return hardLink(srcFile, dstFile, opts.logDebug)
	case srcFile.isSymlink():
		// FIXME: we really need to copy the pass through dest unless they specify otherwise...check the docs
		return copyLink(srcFile, dstFile, opts.logDebug)
	case srcFile.isDir:
		return copyDir(srcFile, dstFile, opts)
	default:
		return copyFile(srcFile, dstFile, opts)
	}
}

// hardLink creates a hard link to src at dst.
func hardLink(src, dst *File, logFunc func(format string, a ...interface{})) error {
	logFunc("creating hard link to src %s at dst %s", src.Path, dst.Path)
	return os.Link(src.Path, dst.Path)
}

// copyLink copies a symbolic link from src to dst.
func copyLink(src, dst *File, logFunc func(format string, a ...interface{})) error {
	logFunc("copying sym link %s to %s", src.Path, dst.Path)
	linkSrc, err := os.Readlink(src.Path)
	if err != nil {
		return err
	}
	return os.Symlink(linkSrc, dst.Path)
}

func copyDir(srcFile, dstFile *File, opts Options) error {
	if !opts.Recursive {
		return errors.Wrapf(ErrOmittingDir, "source directory %s", srcFile.Path)
	}
	if opts.mkdirAll {
		opts.logDebug("making all dirs up to %s", dstFile.Path)
		if err := os.MkdirAll(dstFile.Path, srcFile.fileInfoOnInit.Mode()); err != nil {
			return err
		}
	}

	srcDirEntries, err := ioutil.ReadDir(srcFile.Path)
	if err != nil {
		return errors.Wrapf(ErrReadingSrcDir, "source directory %s: %s", srcFile.Path, err)
	}

	for _, entry := range srcDirEntries {
		newSrc := filepath.Join(srcFile.Path, entry.Name())
		newDst := filepath.Join(dstFile.Path, entry.Name())
		opts.logDebug("recursive cp with src %s and dst %s", newSrc, newDst)
		if err := Copy(
			newSrc,
			newDst,
			opts,
		); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(srcFile, dstFile *File, opts Options) (err error) {
	// shortcut if files are the same file
	if os.SameFile(srcFile.fileInfoOnInit, dstFile.fileInfoOnInit) {
		opts.logDebug("src %s is same file as dst %s", srcFile.Path, dstFile.Path)
		return nil
	}

	// optionally make dst parent directories
	if dstFile.shouldMakeParents(opts) {
		// TODO: permissive perms here to ensure tmp file can write on nix.. ensure we are setting these correctly down the line or fix here
		if err := os.MkdirAll(filepath.Dir(dstFile.Path), 0777); err != nil {
			return err
		}
	}

	if dstFile.existOnInit {
		if dstFile.isDir {
			// optionally append src file name to dst dir like cp does
			if opts.AppendNameToPath {
				dstFile.Path = filepath.Join(dstFile.Path, filepath.Base(srcFile.Path))
				opts.logDebug("because of AppendNameToPath option, setting dst path to %s", dstFile.Path)
			} else {
				return errors.Wrapf(ErrWritingFileToExistingDir, "destination directory %s", dstFile.Path)
			}
		}

		// optionally do not clobber existing dst file
		if opts.NoClobber {
			opts.logDebug("dst %s exists, will not clobber", dstFile.Path)
			return nil
		}

		if opts.Backup != "" {
			if err := backupFile(dstFile, opts.Backup, opts); err != nil {
				return err
			}
		}

	}

	srcFD, err := os.Open(srcFile.Path)
	if err != nil {
		return errors.Wrapf(ErrCannotOpenSrc, "source file %s: %s", srcFile.Path, err)
	}
	defer func() {
		if closeErr := srcFD.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	if opts.Atomic {
		dstDir := filepath.Dir(dstFile.Path)
		tmpFD, err := ioutil.TempFile(dstDir, "copyfile-")
		defer closeAndRemove(tmpFD, opts.logDebug)
		if err != nil {
			return errors.Wrapf(ErrCannotCreateTmpFile, "destination directory %s: %s", dstDir, err)
		}
		opts.logDebug("created tmp file %s", tmpFD.Name())

		//copy src to tmp and cleanup on any error
		opts.logInfo("copying src file %s to tmp file %s", srcFD.Name(), tmpFD.Name())
		if _, err := io.Copy(tmpFD, srcFD); err != nil {
			return err
		}
		if err := tmpFD.Sync(); err != nil {
			return err
		}
		if err := tmpFD.Close(); err != nil {
			return err
		}

		// move tmp to dst
		opts.logInfo("renaming tmp file %s to dst %s", tmpFD.Name(), dstFile.Path)
		if err := os.Rename(tmpFD.Name(), dstFile.Path); err != nil {
			return errors.Wrapf(ErrCannotRenameTempFile, "attempted to rename temp transfer file %s to %s", tmpFD.Name(), dstFile.Path)
		}
	} else {
		dstFD, err := os.Create(dstFile.Path)
		if err != nil {
			return errors.Wrapf(ErrCannotOpenOrCreateDstFile, "destination file %s: %s", dstFile.Path, err)
		}
		defer func() {
			if closeErr := dstFD.Close(); closeErr != nil {
				err = closeErr
			}
		}()

		opts.logInfo("copying src file %s to dst file %s", srcFD.Name(), dstFD.Name())
		if _, err = io.Copy(dstFD, srcFD); err != nil {
			return err
		}
		if err := dstFD.Sync(); err != nil {
			return err
		}
	}

	return setPermissions(dstFile, srcFile.fileInfoOnInit.Mode(), opts)
}

// backupFile will create a backup of the file using the chosen control method.  See Options.Backup.
func backupFile(file *File, control string, opts Options) error {
	// do not copy if the file did not exist
	if !file.existOnInit {
		return nil
	}

	// simple backup
	simple := func() error {
		bkp := file.Path + "~"
		opts.logDebug("creating simple backup file %s", bkp)
		return Copy(file.Path, bkp, opts)
	}

	// next gives the next unused backup file number, 1 above the current highest
	next := func() (int, error) {
		// find general matches that look like numbered backup files
		m, err := filepath.Glob(file.Path + ".~[0-9]*~")
		if err != nil {
			return -1, err
		}

		// get each backup file num substring, convert to int, track highest num
		var highest int
		for _, f := range m {
			subs := numberedBackupFile.FindStringSubmatch(filepath.Base(f))
			if len(subs) > 1 {
				if i, _ := strconv.Atoi(string(subs[1])); i > highest {
					highest = i
				}
			}
		}
		return highest + 1, nil
	}

	// numbered backup
	numbered := func(n int) error {
		return Copy(file.Path, fmt.Sprintf("%s.~%d~", file.Path, n), opts)
	}

	switch control {
	default:
		return errors.Wrapf(ErrInvalidBackupControlValue, "backup value '%s'", control)
	case "off":
		return nil
	case "simple":
		return simple()
	case "numbered":
		i, err := next()
		if err != nil {
			return err
		}
		return numbered(i)
	case "existing":
		i, err := next()
		if err != nil {
			return err
		}

		if i > 1 {
			return numbered(i)
		}
		return simple()
	}
}

func closeAndRemove(file *os.File, logFunc func(format string, a ...interface{})) {
	if file != nil {
		if err := file.Close(); err != nil {
			logFunc("err closing file %s: %s", file.Name(), err)
		}
		if err := os.Remove(file.Name()); err != nil {
			logFunc("err removing file %s: %s", file.Name(), err)
		}
	}
}

// setPermissions will set file level permissions on dst based on options and other criteria.
func setPermissions(dstFile *File, srcMode os.FileMode, opts Options) error {
	var mode os.FileMode
	fi, err := os.Stat(dstFile.Path)
	if err != nil {
		return err
	}
	mode = fi.Mode()

	if dstFile.existOnInit {
		if mode == dstFile.fileInfoOnInit.Mode() {
			opts.logDebug("existing dst %s permissions %s are unchanged", dstFile.Path, mode)
			return nil
		}

		// make sure dst perms are set to their original value
		opts.logDebug("changing dst %s permissions to %s", dstFile.Path, dstFile.fileInfoOnInit.Mode())
		err := os.Chmod(dstFile.Path, dstFile.fileInfoOnInit.Mode())
		if err != nil {
			return errors.Wrapf(ErrCannotChmodFile, "destination file %s: %s", dstFile.Path, err)
		}
	} else {
		if mode == srcMode {
			opts.logDebug("dst %s permissions %s already match src perms", dstFile.Path, mode)
		}

		// make sure dst perms are set to that of src
		opts.logDebug("changing dst %s permissions to %s", dstFile.Path, srcMode)
		err := os.Chmod(dstFile.Path, srcMode)
		if err != nil {
			return errors.Wrapf(ErrCannotChmodFile, "destination file %s: %s", dstFile.Path, err)
		}
	}
	return nil
}

// Options directly represent command line flags associated with GNU file operations.
type Options struct {
	// AppendNameToPath will, when attempting to copy a file to an existing directory, automatically
	// create the file with the same name in the destination directory.  While CP uses this behavior
	// by default it is an assumption better left to the client in a programmatic setting.
	AppendNameToPath bool
	// Atomic will copy contents to a temporary file in the destination's parent directory first, then
	// rename the file to ensure the operation is atomic.
	Atomic bool
	// Backup makes a backup of each existing destination file. The backup suffix is '~'. Acceptable
	// control values are:
	//   - "off"       no backup will be made (default)
	//   - "simple"    always make simple backups
	//   - "numbered"  make numbered backups
	//   - "existing"  numbered if numbered backups exist, simple otherwise
	Backup string
	// Link creates hard links to files instead of copying them.
	Link bool
	// MkdirAll will use os.MkdirAll to create the destination directory if it does not exist, along with
	// any necessary parents.
	MkdirAll bool
	// mkdirAll is an internal tracker for MkdirAll, including other validation checks
	mkdirAll bool
	// NoClobber will not let an existing file be overwritten.
	NoClobber bool
	// Parents will create source directories in dst if they do not already exist. ErrWithParentsDstMustBeDir
	// is returned if destination is not a directory.
	Parents bool
	// Recursive will recurse through sub directories if set true.
	Recursive bool
	// InfoLogFunc will, if defined, handle logging info messages.
	InfoLogFunc func(string)
	// DebugLogFunc will, if defined, handle logging debug messages.
	DebugLogFunc func(string)
}

// setLoggers will configure logging functions, setting noop loggers if log funcs are undefined.
func (o *Options) setLoggers() {
	if o.InfoLogFunc == nil {
		o.InfoLogFunc = func(string) {}
	}
	if o.DebugLogFunc == nil {
		o.DebugLogFunc = func(string) {}
	}
}

// logDebug will log to the DebugLogFunc.
func (o *Options) logDebug(format string, a ...interface{}) {
	o.DebugLogFunc(fmt.Sprintf(format, a...))
}

// logInfo will log to the InfoLogFunc.
func (o *Options) logInfo(format string, a ...interface{}) {
	o.InfoLogFunc(fmt.Sprintf(format, a...))
}
