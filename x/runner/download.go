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

const (
	zipExtension = ".zip"
)

//This should be updated with new mirrors
var knownMirrors = []string{"https://github.com/pokt-network/pocket-core/archive/"}

func DownloadBinary(cfg *types.Config, info *types.UpgradeInfo) error {

	//Get Mirrors Links for Version
	mirrors, err := GetMirrorLinksForVersion(info.Name)

	if err != nil {
		return err
	}

	for _, link := range mirrors {
		//Download File from mirror
		result := DownloadFile(cfg, info, link)
		if result {
			//if successfull break
			break
		}
	}

	_, err = os.Stat(cfg.DownloadCode(info.Name) + zipExtension)
	if os.IsNotExist(err) {
		return err
	}

	//unzip file
	_, err = Unzip(cfg.DownloadCode(info.Name)+zipExtension, cfg.DownloadCode(info.Name))
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	//delete unziped code folder
	defer os.RemoveAll(cfg.DownloadCode(info.Name))

	err = CompilePocketCore(cfg, info)
	if err != nil {
		return err
	}

	return nil
}

func CompilePocketCore(cfg *types.Config, info *types.UpgradeInfo) error {
	//compile binary
	pathToMain := filepath.Join(cfg.DownloadCode(info.Name), "pocket-core-"+info.Name, "app", "cmd", "pocket_core", "main.go")

	cmd := exec.Command("go", "build", "-o", cfg.UpgradeBin(info.Name), pathToMain)
	stdout, err := cmd.Output()

	if err != nil {
		errors.Wrap(err, "Error building")
		return err
	}
	fmt.Print(string(stdout))

	_, err = os.Stat(cfg.UpgradeBin(info.Name))
	if os.IsNotExist(err) {
		return errors.Wrapf(err, "upgrade %s not found", info.Name)
	}

	return nil
}

func DownloadFile(cfg *types.Config, info *types.UpgradeInfo, link string) bool {

	path := cfg.DownloadCode(info.Name)

	// Create the file
	// this will fail if ../release/code doesnt exist
	out, err := os.Create(path + zipExtension)
	if err != nil {
		return false
	}
	defer out.Close()

	resp, err := http.Get(link)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

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
