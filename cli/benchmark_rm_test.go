package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DataDrake/cli-ng/cmd"
)

// This function tests the performance of a large remove operation by adding
// everything in your documents folder and removing it. The time the rm itself
// took is printed.
func BenchmarkBigRm(b *testing.B) {
	u, _ := os.UserHomeDir()
	testRoot := filepath.Join(u, "Documents/")
	_ = os.Chdir(testRoot)
	_ = os.RemoveAll(testRoot + "/.ait")
	InitRun(nil, nil)
	addArgs := &cmd.CMD{
		Args: &AddArgs{Paths: []string{"."}},
	}
	AddRun(nil, addArgs)
	files, _ := ioutil.ReadDir(testRoot)
	var fileNames []string
	for _, fi := range files {
		fileNames = append(fileNames, fi.Name())
	}
	rmArgs := &cmd.CMD{
		Args: &RemoveArgs{Paths: fileNames},
		Flags: &RemoveFlags{All: false},
	}
	start := time.Now()
	RemoveRun(nil, rmArgs)
	fmt.Println("\n\t******** Rm all folders took",
		time.Since(start).Milliseconds(), "ms ********\n ")
	_ = os.RemoveAll(testRoot + "/.ait")
}

func BenchmarkRmExtensions(b *testing.B) {
	u, _ := os.UserHomeDir()
	testRoot := filepath.Join(u, "Documents/")
	_ = os.Chdir(testRoot)
	_ = os.RemoveAll(testRoot + "/.ait")
	InitRun(nil, nil)
	addArgs := &cmd.CMD{
		Args: &AddArgs{Paths: []string{"."}},
	}
	AddRun(nil, addArgs)
	ext := "java,c,json,md,js"
	rmArgs := &cmd.CMD{
		Flags: &RemoveFlags{
			All:        false,
			Extensions: ext,
		},
	}
	start := time.Now()
	RemoveRun(nil, rmArgs)
	fmt.Println("\n\t******** Rm", ext, "took",
		time.Since(start).Milliseconds(), "ms ********\n ")
	_ = os.RemoveAll(testRoot + "/.ait")
}
