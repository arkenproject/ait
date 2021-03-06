package keysets

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/arken/ait/config"
	"github.com/arken/ait/ipfs"
	"github.com/arken/ait/types"
	"github.com/arken/ait/utils"

	"github.com/schollz/progressbar/v3"
)

const delimiter = "  "

// Generate is the public facing function for the creation of a keyset file.
// Depending on the value of overwrite, the keyset file is either generated from
// scratch or added to.
func Generate(path string, overwrite bool) error {
	if overwrite {
		return createNew(path)
	}
	return amendExisting(path)
}

// createNew creates a keyset file with the given path. Path should not be the
// desired directory, rather it should be a full path to a file which does not
// exist yet (will be truncated if it does exist), and the file should end in
// ".ks" The resultant keyset files contains the name (not path) of the file and
// an IPFS cid hash, separated by a space.
func createNew(path string) error {
	_ = os.MkdirAll(filepath.Dir(path), os.ModePerm)

	keySetFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	addedFiles, err := os.OpenFile(utils.AddedFilesPath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}

	// In order to not copy files to ~/.ait/ipfs/ we need to create a workdir symlink
	// in .ait
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	link := filepath.Join(filepath.Dir(config.Global.IPFS.Path), "workdir")
	err = os.Symlink(wd, link)
	if err != nil {
		if strings.HasSuffix(err.Error(), "file exists") {
			os.Remove(link)
			err = os.Symlink(wd, link)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	contents := types.NewSortedStringSet()
	utils.FillSet(contents, addedFiles)
	addedFiles.Close()

	// For large Datasets display a loading bar.
	var ipfsBar *progressbar.ProgressBar
	barPresent := false
	if contents.Size() > 30 {
		fmt.Println("Adding Files to Embedded IPFS Node:")
		ipfsBar = progressbar.Default(int64(contents.Size()))
		ipfsBar.RenderBlank()
		barPresent = true
	}

	var output strings.Builder
	output.Grow(contents.Size())

	err = contents.ForEach(func(filePath string) error {
		linkPath := filepath.Join(link, filePath)
		fmt.Fprintf(&output, "%s\n", getKeySetLineFromPath(linkPath))
		if barPresent {
			ipfsBar.Add(1)
		}
		return nil
	})
	_, err = keySetFile.WriteString(output.String())
	if err != nil {
		cleanup(keySetFile)
		os.Remove(link)
		return err
	}
	err = os.Remove(link)
	if err != nil {
		return err
	}
	err = keySetFile.Close()
	if err != nil {
		return err
	}
	return err
}

// amendExisting looks at current files in added_files and adds any that aren't
// already in the keyset file to the keyset files. The keyset file in question
// should be at path.
func amendExisting(ksPath string) error {
	doneChan := make(chan int, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)

	// Display Spinner on amend.
	go utils.SpinnerWait(doneChan, "Reading Previous Keyset File...", &wg)

	keySetFile, err := os.OpenFile(ksPath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer keySetFile.Close()
	addedFiles, err := os.OpenFile(utils.AddedFilesPath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer addedFiles.Close()
	addedFilesContents := make(map[string]string)
	// ^ map of cid -> filePATH
	fillMapWithCID(addedFilesContents, addedFiles)
	ksContents := make(map[string]string)
	// ^ map of cid -> fileNAME
	fillMapWithCID(ksContents, keySetFile)
	newFiles := make(map[string]string)

	doneChan <- 0
	wg.Wait()

	// For large Datasets display a loading bar.
	var namesBar *progressbar.ProgressBar
	barPresent := false
	if len(addedFilesContents) > 30 {
		fmt.Println("Reading File Names:")
		namesBar = progressbar.Default(int64(len(addedFilesContents)))
		namesBar.RenderBlank()
		barPresent = true
	}

	// ^ paths of the files which will be added
	for cid, path := range addedFilesContents {
		if _, contains := ksContents[cid]; !contains {
			filename := strings.Join(strings.Fields(filepath.Base(path)), "-")
			newFiles[cid] = filename
		} else {
			delete(ksContents, cid)
		}
		if barPresent {
			namesBar.Add(1)
			namesBar.RenderBlank()
		}
	}

	var ipfsBar *progressbar.ProgressBar
	if barPresent {
		fmt.Println("Adding Files to Embedded IPFS Node:")
		ipfsBar = progressbar.Default(int64(len(newFiles)))
		ipfsBar.RenderBlank()
	}

	for cid, filename := range newFiles {
		line := getKeySetLine(filename, cid)
		_, err := keySetFile.WriteString(line + "\n")
		if err != nil {
			return err
		}
		if barPresent {
			ipfsBar.Add(1)
		}
	}
	return nil
}

// cleanup closes and deletes the given file.
func cleanup(file *os.File) {
	path := file.Name()
	file.Close()
	_ = os.Remove(path)
}

// getKeySetLine returns a properly formed line for a KeySet file given a path
// to a file. No newline at the end.
func getKeySetLineFromPath(filePath string) string {
	// Scrub filename for spaces and replace with dashes.
	cid, err := ipfs.Add(filePath, true)
	utils.CheckErrorWithCleanup(err, utils.SubmissionCleanup)
	filename := strings.Join(strings.Fields(filepath.Base(filePath)), "-")
	return getKeySetLine(filename, cid)
}

// getKeySetLine returns a properly formed line for a KeySet file. It expects a
// fileNAME (not path) and an IPFS cid. No newline at the end.
func getKeySetLine(filename, cid string) string {
	// Scrub filename for spaces and replace with dashes.
	filename = strings.Join(strings.Fields(filename), "-")
	return cid + delimiter + filename
}

// fillMapWithCID will fill the given map with IPFS cid hashes as the key and
// either the filename or filepath as the value. This function ONLY be used for
// files that are standard keyset files or files that are just newline separated
// paths. Returns the length of the longest fileNAME, not path. If the file was
// a keyset file, the values are filenames. If the file was just file paths, the
// the values will be file paths.
func fillMapWithCID(contents map[string]string, file *os.File) {
	// In order to not copy files to ~/.ait/ipfs/ we need to create a workdir symlink
	// in .ait
	wd, err := os.Getwd()
	if err != nil {
		utils.FatalWithCleanup(utils.SubmissionCleanup, err.Error())
	}
	link := filepath.Join(filepath.Dir(config.Global.IPFS.Path), "workdir")
	err = os.Symlink(wd, link)
	if err != nil {
		if strings.HasSuffix(err.Error(), "file exists") {
			os.Remove(link)
			err = os.Symlink(wd, link)
			if err != nil {
				utils.FatalWithCleanup(utils.SubmissionCleanup, err.Error())
			}
		} else {
			utils.FatalWithCleanup(utils.SubmissionCleanup, err.Error())
		}
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	if filepath.Ext(file.Name()) == ".ks" {
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) > 0 {
				pair := strings.Fields(line)
				if len(pair) != 2 {
					utils.FatalWithCleanup(utils.SubmissionCleanup,
						"Malformed KeySet file detected:", file.Name())
				}
				contents[pair[0]] = pair[1]
			}
		}
	} else {
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) > 0 {
				filename := filepath.Base(line)
				cid, err := ipfs.Add(filepath.Join(link, line), true)
				utils.CheckErrorWithCleanup(err, utils.SubmissionCleanup)
				contents[cid] = filename
			}
		}
	}
	os.Remove(link)
}
