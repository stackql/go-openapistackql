package semver

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/Masterminds/semver"
)

const (
	stableSemVerRegexStr string = `^v[0-9]+(\.[0-9]+){0,2}$`
)

var (
	_ *regexp.Regexp = regexp.MustCompile(stableSemVerRegexStr)
)

func FindLatestStable(svSlice []string) (string, error) {
	var versionsAvailable []*semver.Version
	for _, e := range svSlice {
		nv, err := semver.NewVersion(e)
		if err != nil {
			return "", err
		}
		versionsAvailable = append(versionsAvailable, nv)
	}
	sort.Sort(semver.Collection(versionsAvailable))
	if len(versionsAvailable) == 0 {
		return "", fmt.Errorf("no versions available")
	}
	return versionsAvailable[len(versionsAvailable)-1].Original(), nil
}

func FindLatest(svSlice []string) (string, error) {
	var versionsAvailable []*semver.Version
	for _, e := range svSlice {
		nv, err := semver.NewVersion(e)
		if err != nil {
			return "", err
		}
		versionsAvailable = append(versionsAvailable, nv)
	}
	sort.Sort(semver.Collection(versionsAvailable))
	if len(versionsAvailable) == 0 {
		return "", fmt.Errorf("no versions available")
	}
	return versionsAvailable[len(versionsAvailable)-1].Original(), nil
}
