package media

import (
	"mime"
	"net/http"
	"regexp"

	"github.com/stackql/go-openapistackql/pkg/fuzzymatch"
)

const (
	MediaTypeHTML        string = "text/html"
	MediaTypeJson        string = "application/json"
	MediaTypeScimJson    string = "application/scim+json"
	MediaTypeOctetStream string = "application/octet-stream"
	MediaTypeTextPlain   string = "text/plain"
	MediaTypeXML         string = "application/xml"
	MediaTypeTextXML     string = "text/xml"
)

var (
	synonymJSONRegexp        *regexp.Regexp                  = regexp.MustCompile(`^application/[\S]*json[\S]*$`)
	synonymXMLRegexp         *regexp.Regexp                  = regexp.MustCompile(`^(?:application|text)/[\S]*xml[\S]*$`)
	DefaultMediaFuzzyMatcher fuzzymatch.FuzzyMatcher[string] = fuzzymatch.NewRegexpStringMetcher(
		[]fuzzymatch.StringFuzzyPair{
			fuzzymatch.NewFuzzyPair(synonymJSONRegexp, MediaTypeJson),
			fuzzymatch.NewFuzzyPair(synonymXMLRegexp, MediaTypeXML),
		})
)

func IsXMLSynonym(mediaType string) bool {
	return isXMLSynonym(mediaType)
}

func IsJSONSynonym(mediaType string) bool {
	return isJSONSynonym(mediaType)
}

func isXMLSynonym(mediaType string) bool {
	return synonymXMLRegexp.MatchString(mediaType)
}

func isJSONSynonym(mediaType string) bool {
	return synonymJSONRegexp.MatchString(mediaType)
}

func IsAcceptableMediaType(mediaType string) bool {
	return isAcceptableMediaType(mediaType)
}

func isAcceptableMediaType(mediaType string) bool {
	switch mediaType {
	case MediaTypeHTML,
		MediaTypeJson,
		MediaTypeScimJson,
		MediaTypeOctetStream,
		MediaTypeTextPlain,
		MediaTypeXML:
		return true
	default:
		return false
	}
}

func GetResponseMediaType(r *http.Response, defaultMediaType string) (string, error) {
	rt := r.Header.Get("Content-Type")
	var mediaType string
	var err error
	if rt != "" {
		mediaType, _, err = mime.ParseMediaType(rt)
		if err != nil {
			return "", err
		}
		return mediaType, nil
	}
	return "", nil
}
