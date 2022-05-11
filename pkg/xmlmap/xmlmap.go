package xmlmap

import (
	"io"

	mxj "github.com/clbanning/mxj/v2"
)

var _ mxj.Map

func Unmarshal(xmlReader io.ReadCloser) (map[string]interface{}, error) {
	mv, err := mxj.NewMapXmlReader(xmlReader)
	if err != nil {
		return nil, err
	}
	return mv, nil
}
