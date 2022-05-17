package openapistackql

import (
	"mime"
	"net/http"
)

const (
	MediaTypeHTML        string = "text/html"
	MediaTypeJson        string = "application/json"
	MediaTypeOctetStream string = "application/octet-stream"
	MediaTypeTextPlain   string = "text/plain"
	MediaTypeXML         string = "application/xml"
	MediaTypeTextXML     string = "text/xml"
)

func IsAcceptableMediaType(mediaType string) bool {
	return isAcceptableMediaType(mediaType)
}

func isAcceptableMediaType(mediaType string) bool {
	switch mediaType {
	case MediaTypeHTML,
		MediaTypeJson,
		MediaTypeOctetStream,
		MediaTypeTextPlain,
		MediaTypeXML:
		return true
	default:
		return false
	}
}

func getResponseMediaType(r *http.Response) (string, error) {
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
