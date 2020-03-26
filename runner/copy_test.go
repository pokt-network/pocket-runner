package runner

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

const debug = true

func TestFileCopyOnDstWithInvalidPermissionsReturnsNoErrorWhenAtomic(t *testing.T) {
	// create and write to source inFile
	src := tmpFile()
	content := []byte("foo")

	if ioutil.WriteFile(src, content, 0644) != nil {
		t.Errorf("error writing file with perms 0644")
		t.FailNow()
	}

	dst := tmpFile()
	// explicitly set our dst inFile perms so that we cannot copy
	if os.Chmod(dst, 0040) != nil {
		t.Errorf("error setting perms 0040")
		t.FailNow()
	}

	if Copy(src, dst, Options{Atomic: true}) != nil {
		t.Errorf("could not copy src into dst")
		t.FailNow()
	}

	// make sure we can read out the correct content
	if os.Chmod(dst, 0655) != nil {
		t.Errorf("error setting perms 0655")
		t.FailNow()
	}
	b, err := ioutil.ReadFile(dst)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if !bytes.Equal(content, b) {
		t.Errorf("got %v, want %v", b, content)
	}
}

func TestFileIsSymlink(t *testing.T) {
	old := tmpFile()
	new := tmpFilePathUnused()
	if os.Symlink(old, new) != nil {
		t.Errorf("not a SymLink")
		t.FailNow()
	}

	newFileInfo, err := os.Lstat(new)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	f := File{
		Path:           new,
		fileInfoOnInit: newFileInfo,
	}
	if !f.isSymlink() {
		t.Error("Not a SymLink")
		t.FailNow()
	}
}

func TestIsSymlinkFailsWithRegularFile(t *testing.T) {
	tmp := tmpFile()
	f := NewFile(tmp)
	f.setInfo()
	if f.isSymlink() {
		t.Error("Is a SymLink")
		t.FailNow()
	}
}

func TestPermissionsAfterCopy(t *testing.T) {
	tests := []struct {
		name             string
		atomic           bool
		dstShouldExist   bool
		srcPerms         os.FileMode
		expectedDstPerms os.FileMode
	}{
		{
			name:             "preserve_src_perms_when_dst_not_exist_0655",
			dstShouldExist:   false,
			srcPerms:         os.FileMode(0655),
			expectedDstPerms: os.FileMode(0655),
		},
		{
			name:             "preserve_src_perms_when_dst_not_exist_0777",
			dstShouldExist:   false,
			srcPerms:         os.FileMode(0777),
			expectedDstPerms: os.FileMode(0777),
		},
		{
			name:             "preserve_src_perms_when_dst_not_exist_0741",
			dstShouldExist:   false,
			srcPerms:         os.FileMode(0741),
			expectedDstPerms: os.FileMode(0741),
		},
		{
			name:             "preserve_dst_perms_when_dst_exists_0654",
			dstShouldExist:   true,
			srcPerms:         os.FileMode(0655),
			expectedDstPerms: os.FileMode(0654),
		},
		{
			name:             "preserve_dst_perms_when_dst_exists_0651",
			dstShouldExist:   true,
			srcPerms:         os.FileMode(0655),
			expectedDstPerms: os.FileMode(0651),
		},
		{
			name:             "preserve_dst_perms_when_dst_exists_0777",
			dstShouldExist:   true,
			srcPerms:         os.FileMode(0655),
			expectedDstPerms: os.FileMode(0777),
		},
		{
			name:             "preserve_dst_perms_when_dst_exists_0666",
			atomic:           false,
			dstShouldExist:   true,
			srcPerms:         os.FileMode(0655),
			expectedDstPerms: os.FileMode(0666),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := tmpFile()
			if err := os.Chmod(src, tt.srcPerms); err != nil {
				t.Error(err)
				t.FailNow()
			}

			var dst string
			if tt.dstShouldExist {
				dst = tmpFile()
				// set dst perms to ensure they are distinct beforehand
				if err := os.Chmod(dst, tt.expectedDstPerms); err != nil {
					t.Error(err)
					t.FailNow()
				}

			} else {
				dst = tmpFilePathUnused()
			}

			if err := Copy(src, dst, Options{
				Atomic:      tt.atomic,
				InfoLogFunc: infoLogger,
			}); err != nil {
				t.Error(err)
				t.FailNow()
			}

			// check our perms
			d, err := os.Stat(dst)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}
			dstPerms := d.Mode()
			if fmt.Sprint(tt.expectedDstPerms) != fmt.Sprint(dstPerms) {
				t.Errorf("got %s, want %s", dstPerms, tt.expectedDstPerms)
			}
		})
	}
}

func TestCopyingSymLinks(t *testing.T) {
	src := tmpFile()
	content := []byte("foo")

	if err := ioutil.WriteFile(src, content, 0655); err != nil {
		t.Error(err)
		t.FailNow()
	}
	srcSym := tmpFilePathUnused()
	if err := os.Symlink(src, srcSym); err != nil {
		t.Error(err)
		t.FailNow()
	}

	dstSym := tmpFilePathUnused()

	// copy sym link
	if err := SimpleCopy(srcSym, dstSym); err != nil {
		t.Error(err)
		t.FailNow()
	}

	// verify that dst is a sym link
	sfi, err := os.Lstat(dstSym)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if sfi.Mode()&os.ModeSymlink == 0 {
		t.Error("Is not a Symlink")
		t.FailNow()
	}

	// verify content is the same in symlinked file
	b, err := ioutil.ReadFile(dstSym)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if !bytes.Equal(content, b) {
		t.Errorf("got %s, want %s", b, content)
		t.FailNow()
	}
}

func TestCreatingHardLinksWithLinkOpt(t *testing.T) {
	tests := []struct {
		name string
		src  string
		dst  string
		opts Options
	}{
		{
			name: "absent_dst",
			src:  tmpFile(),
			dst:  tmpFilePathUnused(),
			opts: Options{Link: true},
		},
		//{  // TODO setup when force is implemented
		//	name: " existing_dst",
		//	src: tmpFile(),
		//	dst: tmpFile(),
		//	opts: Options{Link: true, Force: true},
		//},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := []byte("foo")
			if err := ioutil.WriteFile(tt.src, content, 0655); err != nil {
				t.Error(err)
				t.FailNow()
			}

			if err := Copy(tt.src, tt.dst, tt.opts); err != nil {
				t.Error(err)
				t.FailNow()
			}

			sFI, err := os.Stat(tt.src)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}
			dFI, err := os.Stat(tt.dst)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}
			if !os.SameFile(sFI, dFI) {
				t.Errorf("got %s, want %s", dFI, sFI)
			}
		})
	}
}

// memoizeTmpDir holds memoization info for the temporary directory
var memoizeTmpDir string

// unusedFileNum tracks a unique file identifier
var unusedFileNum int

// unusedDirNum tracks a unique dir identifier
var unusedDirNum int

// tmpDirPath gets the path of a temporary directory on the system
func tmpDirPath() string {
	// memoize
	if memoizeTmpDir != "" {
		return memoizeTmpDir
	}

	d, _ := ioutil.TempDir("", "")
	return d
}

// tmpDirPathUnused returns the path for a temp directory that does not exist yet
func tmpDirPathUnused() string {
	d, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}

	for {
		d = filepath.Join(d, fmt.Sprintf("%s%d", "dir", unusedDirNum))
		// we expect to see an error if the dir path is unused
		if _, err := os.Stat(d); err == nil {
			// bump file number if the file created with that number exists
			unusedDirNum += 1
		} else {
			return d
		}
	}
}

// tmpFile creates a new, empty temporary file and returns the full path
func tmpFile() string {
	src, err := ioutil.TempFile("", "*.txt")
	if err != nil {
		panic(fmt.Sprintf("temp file creation failed: %s", err))
	}
	defer func() {
		_ = src.Close()
	}()

	return src.Name()
}

// tmpFilePathUnused returns the path for a temp file that does not yet exist
func tmpFilePathUnused() string {
	d, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}

	// derive file name of potentially unused file
	tmpFile := func() string {
		return path.Join(d, fmt.Sprintf("%s%d", "file", unusedFileNum))
	}

	for {
		// we expect to see an error if the file path is unused
		if _, err := os.Stat(tmpFile()); err == nil {
			// bump file number if the file created with that number exists
			unusedFileNum += 1
		} else {
			return tmpFile()
		}
	}
}

func errContains(err error, substring string) bool {
	errString := fmt.Sprintf("%s", err)
	if strings.Contains(errString, substring) {
		return true
	}
	fmt.Println("error:", errString)
	fmt.Println("substring:", substring)
	return false
}

func infoLogger(msg string) {
	if debug {
		log.Println(msg)
	}
}
