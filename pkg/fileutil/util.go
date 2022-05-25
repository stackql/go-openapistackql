package fileutil

import (
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func GetFilePathFromRepositoryRoot(relativePath string) (string, error) {
	rv, err := getFilePathUnescapedFromRepositoryRoot(relativePath)
	return strings.ReplaceAll(rv, `\`, `\\`), err
}

func GetFilePathUnescapedFromRepositoryRoot(relativePath string) (string, error) {
	return getFilePathUnescapedFromRepositoryRoot(relativePath)
}

func getFilePathUnescapedFromRepositoryRoot(relativePath string) (string, error) {
	_, filename, _, _ := runtime.Caller(0)
	curDir := filepath.Dir(filename)
	return filepath.Abs(filepath.Join(curDir, "../..", relativePath))
}

func GetForwardSlashFilePathFromRepositoryRoot(relativePath string) (string, error) {
	_, filename, _, _ := runtime.Caller(0)
	curDir := path.Dir(filename)
	rv, err := filepath.Abs(path.Join(curDir, "../..", relativePath))
	return filepath.ToSlash(rv), err
}

// func FilePathJoin(paths ...string) string {
// 	return strings.ReplaceAll(filepath.Join(paths...), `\`, `\\`)
// }
