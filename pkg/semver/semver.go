package semver

import (
	"regexp"
	"sort"

	"github.com/Masterminds/semver"
)

const (
	stableSemVerRegexStr string = `^v[0-9]+(\.[0-9]+){0,2}$`
)

var (
	stableSemVerRegex *regexp.Regexp = regexp.MustCompile(stableSemVerRegexStr)
)

func FindLatestStable(versions []string) (string, error) {
	var acceptableVersions []*semver.Version
	var allVersions []*semver.Version
	for _, v := range versions {
		sv, err := semver.NewVersion(v)
		if err != nil {
			return "", err
		}
		allVersions = append(allVersions, sv)
		if stableSemVerRegex.MatchString(v) {
			acceptableVersions = append(acceptableVersions, sv)
		}
	}
	if len(acceptableVersions) > 0 {
		return "v" + getMaxFromCollection(semver.Collection(acceptableVersions)), nil
	}
	return "v" + getMaxFromCollection(semver.Collection(allVersions)), nil
}

func getMaxFromCollection(c semver.Collection) string {
	sort.Sort(c)
	return c[c.Len()-1].String()
}
