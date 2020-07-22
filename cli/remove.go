package cli

import (
	"bufio"
	"errors"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/DataDrake/cli-ng/cmd"
)

// Remove is the reverse of the add method. Given a set of file patterns, it
// un-stages all files that match any of the patterns. It also takes a special arg
// "--all" which will un-stage all files currently staged. Currently, this is
// the same behavior as if it was passed "." or "*"
// To remove lines from added_files, this function creates a temporary file which
// holds all the lines that will stay from added_files, and at the end, it deletes
// the original added_files and renames the temp file to be the new added_files.
var Remove = cmd.CMD{
	Name:  "remove",
	Alias: "rm",
	Short: "Remove a file or directory from AIT's tracked files.",
	Args:  &RemoveArgs{},
	Flags: &RemoveFlags{},
	Run:   RemoveRun,
}

// RemoveArgs handles the specific arguments for the remove command.
type RemoveArgs struct {
	Patterns []string
}

// RemoveFlags handles the specific flags for the remove command.
type RemoveFlags struct {
	All bool `long:"all" desc:"remove all staged files."`
}

// RemoveRun executes the remove function.
//TODO: rewrite this function so that it actually works
func RemoveRun(_ *cmd.RootCMD, c *cmd.CMD) {
	flags := c.Flags.(*RemoveFlags)
	args := c.Args.(*RemoveArgs)

	if !FileExists(addedFilesPath) || GetFileSize(addedFilesPath) == 0 {
		log.Fatal(errors.New("no files currently staged, nothing was done"))
	}
	if flags.All || args.Patterns[0] == "." {
		file, err := os.OpenFile(addedFilesPath, os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		file.Close()
		return
	}
	addedFiles, err := os.OpenFile(addedFilesPath, os.O_RDONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	tempFile, err := os.OpenFile(".ait/temp", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(addedFiles)
	scanner.Split(bufio.ScanLines)
	contents := make(map[string]struct{})
	for scanner.Scan() {
		for _, pattern := range args.Patterns {
			pattern = filepath.Clean(pattern)
			m, _ := path.Match(pattern, scanner.Text())
			_, contains := contents[scanner.Text()]
			if !(m || contains) {
				contents[scanner.Text()] = struct{}{}
				_, err := tempFile.WriteString(scanner.Text() + "\n")
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
	tempFile.Close()
	addedFiles.Close()
	err = os.Remove(addedFilesPath)
	if err != nil {
		log.Fatal(err)
	}
	err = os.Rename(".ait/temp", addedFilesPath)
	if err != nil {
		log.Fatal(err)
	}
}
