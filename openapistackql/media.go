package openapistackql

const (
	MediaTypeHTML        string = "text/html"
	MediaTypeJson        string = "application/json"
	MediaTypeOctetStream string = "application/octet-stream"
	MediaTypeTextPlain   string = "text/plain"
	MediaTypeXML         string = "application/xml"
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
