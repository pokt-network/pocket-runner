package runner

import (
	"archive/zip"
	"fmt"
	"github.com/pkg/errors"
	"github.com/pokt-network/pocket-runner/internal/types"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var knownMirrors = []string{"https://github.com/pokt-network/pocket-core/archive/"}

// DoUpgrade will be called after the log message has been parsed and the process has terminated.
// We can now make any changes to the underlying directory without interferance and leave it
// in a state, so we can make a proper restart
func Upgrade(cfg *types.Config, info *types.UpgradeInfo) error {
	err := types.CheckBinary(cfg.UpgradeBin(info.Name))

	// Simplest case is to switch the link
	if err != nil {
		return errors.Wrapf(err, "No binary available for upgrade")
	}
	// we have the binary - do it
	return cfg.SetCurrentUpgrade(info.Name)
}

func DownloadBinary(cfg *types.Config, info *types.UpgradeInfo) error {

	//1st Get Mirrors Links
	mirrors, err := GetMirrorLinksForVersion(info.Version)

	if err != nil {
		return err
	}

	for _, link := range mirrors {

		//Download File from Link
		result := DownloadFile(cfg, info, link)

		if result {
			//if successfull break
			break
		}
	}

	_, err = os.Stat(cfg.DownloadCode(info.Name) + ".zip")
	if os.IsNotExist(err) {

		return err
	}

	//unzip file
	Unzip(cfg.DownloadCode(info.Name)+".zip", cfg.DownloadCode(info.Name))

	//compile binary

	pathToMain := filepath.Join(cfg.DownloadCode(info.Name), "pocket-core-"+info.Name, "app", "cmd", "pocket_core", "main.go")

	cmd := exec.Command("go", "build", "-o", cfg.UpgradeBin(info.Name), pathToMain)
	stdout, err := cmd.Output()

	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Print(string(stdout))

	return nil
}

func DownloadFile(cfg *types.Config, info *types.UpgradeInfo, link string) bool {

	path := cfg.DownloadCode(info.Name)

	// Create the file
	out, err := os.Create(path + ".zip")
	if err != nil {
		return false
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(link)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return false
	}

	return true

}

func GetMirrorLinksForVersion(version string) ([]string, error) {

	versionLinks := make([]string, 0, 10)

	if version != "" {

		for _, mirror := range knownMirrors {

			versionLinks = append(versionLinks, mirror+version+".zip")
		}

		return versionLinks, nil
	} else {
		return versionLinks, errors.New("No Version")
	}

}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
