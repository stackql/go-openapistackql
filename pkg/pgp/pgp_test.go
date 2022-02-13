package pgp_test

import (
	"testing"

	"github.com/stackql/go-openapistackql/pkg/fileutil"
	// . "github.com/stackql/go-openapistackql/pkg/pgp"

	"gotest.tools/assert"
)

func TestPgpSigningDefaulted(t *testing.T) {

	_, err := fileutil.GetFilePathFromRepositoryRoot("test/pgp/some-sample-file.txt")
	assert.NilError(t, err)

}
