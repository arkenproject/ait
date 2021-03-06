package github

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/arken/ait/utils"

	"github.com/google/go-github/v32/github"
)

// CreateFile attempts to upload the file at localPath to the current repo at
// the path repoPath.
func CreateFile(localPath, repoPath, commit string, isPR bool) {
	file, err := ioutil.ReadFile(localPath)
	utils.CheckError(err)
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(commit),
		Content: file,
	}
	owner := cache.upstream.owner
	if isPR {
		owner = *cache.user.Login
		// if it's a PR, the repo belongs to our user and not what we pulled out
		// of the original URL.
	}
	_, _, err = client.Repositories.CreateFile(cache.ctx, owner, cache.upstream.name,
		repoPath, opts)
	utils.CheckError(err)
}

// UpdateFile attempts to upload the file at localPath to the current repo at
// the path repoPath. The file is expected to exist in the repo.
func UpdateFile(localPath, repoPath, commit string, isPR bool) {
	file, err := ioutil.ReadFile(localPath)
	utils.CheckError(err)
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(commit),
		Content: file,
		SHA:     github.String(getFileSHA(repoPath, isPR)),
	}
	owner := cache.upstream.owner
	if isPR {
		owner = *cache.user.Login
	}
	_, _, err = client.Repositories.UpdateFile(cache.ctx, owner, cache.upstream.name,
		repoPath, opts)
	utils.CheckError(err)
}

// ReplaceFile attempts to upload the file at localPath to the current repo at
// the path repoPath. The file is expected to exist in the repo. It deletes the
// old version and uploads the new one.
func ReplaceFile(localPath, repoPath, commit string, isPR bool) {
	file, err := ioutil.ReadFile(localPath)
	utils.CheckError(err)
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(commit),
		Content: file,
		SHA:     github.String(getFileSHA(repoPath, isPR)),
	}
	owner := cache.upstream.owner
	if isPR {
		owner = *cache.user.Login
	}
	_, _, err = client.Repositories.DeleteFile(cache.ctx, owner, cache.upstream.name,
		repoPath, opts)
	utils.CheckError(err)
	opts.SHA = nil
	_, _, err = client.Repositories.CreateFile(cache.ctx, owner, cache.upstream.name,
		repoPath, opts)
}

// getFileSHA returns the sha of a file in the current repo. Returns "" if the
// file doesn't exist. path should be the path to the file in the repo, not
// locally
func getFileSHA(path string, isPR bool) string {
	owner := cache.upstream.owner
	if isPR {
		owner = *cache.user.Login
	}
	sha, ok := cache.shas[path]
	if ok && sha != "" {
		return sha
	}
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	opts := &github.RepositoryContentGetOptions{}
	_, contents, resp, err := client.Repositories.GetContents(cache.ctx, owner,
		cache.upstream.name, dir, opts)
	if err != nil {
		if resp != nil && (resp.Response.StatusCode == 404 || resp.Response.StatusCode == 403) {
			return ""
		}
		utils.FatalPrintln(err)
	}
	for _, file := range contents {
		// fetch the metadata of all the files in the directory the keyset file
		// is supposed to go into.
		if *file.Name == base {
			cache.shas[path] = *file.SHA
			return *file.SHA
		}
	}
	return "" //if the file didn't exist return empty string
}

// KeysetExistsInRepo returns true if the file at path exists in repo, false
// otherwise. isPR is to know whether to check the upstream or the fork.
func KeysetExistsInRepo(path string, isPR bool) bool {
	return getFileSHA(path, isPR) != ""
}

// DownloadRepoAppTemplate looks for a file called "application.md" in the root
// of the repo and downloads it if such a file exists.
func DownloadRepoAppTemplate() (string, error) {
	path := filepath.Join(".ait", cache.upstream.name+"_application.md")
	return path, DownloadFile("application.md", path)
}

// DownloadFile downloads the file at repoPath from the upstream repository to
// the given localPath
func DownloadFile(repoPath, localPath string) error {
	if ok, _ := utils.IsWithinRepo(localPath); ok {
		dir := filepath.Dir(localPath)
		err := os.MkdirAll(dir, 0751)
		if err != nil {
			return err
		}
	}
	opts := &github.RepositoryContentGetOptions{}
	reader, err := client.Repositories.DownloadContents(cache.ctx, cache.upstream.owner,
		cache.upstream.name, repoPath, opts)
	if err != nil {
		return err //probably means the file didn't exist in the fork
	}
	data, err := ioutil.ReadAll(reader)
	utils.CheckError(err)
	err = ioutil.WriteFile(localPath, data, 0644)
	reader.Close()
	return err
}
