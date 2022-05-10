package openapistackql

import (
	"encoding/json"
	"encoding/xml"
	"io"
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

func marshalResponse(r *http.Response) (interface{}, error) {
	body := r.Body
	if body != nil {
		defer body.Close()
	} else {
		return nil, nil
	}
	var target interface{}
	mediaType, err := getResponseMediaType(r)
	if err != nil {
		return nil, err
	}
	switch mediaType {
	case MediaTypeJson:
		err = json.NewDecoder(body).Decode(&target)
	case MediaTypeXML, MediaTypeTextXML:
		err = xml.NewDecoder(body).Decode(&target)
	case MediaTypeOctetStream:
		target, err = io.ReadAll(body)
	case MediaTypeTextPlain, MediaTypeHTML:
		var b []byte
		b, err = io.ReadAll(body)
		if err == nil {
			target = string(b)
		}
	default:
		target, err = io.ReadAll(body)
	}
	return target, err
}
