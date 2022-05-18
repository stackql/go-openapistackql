package xmlmap_test

import (
	"bytes"
	"io"
	"path"
	"path/filepath"
	"testing"

	"gotest.tools/assert"

	"github.com/stackql/go-openapistackql/pkg/fileutil"
	. "github.com/stackql/go-openapistackql/pkg/xmlmap"

	"github.com/getkin/kin-openapi/openapi3"
)

func getFileRoot(t *testing.T) string {
	rv, err := fileutil.GetFilePathUnescapedFromRepositoryRoot(path.Join("test", "registry", "src"))
	assert.NilError(t, err)
	return rv
}

func TestListVolumesSingle(t *testing.T) {

	m, err := Unmarshal(getAwsEc2ListSingleResponseReader())
	assert.NilError(t, err)
	assert.Assert(t, m != nil)
}

func TestListVolumesMulti(t *testing.T) {

	m, err := GetSubObjArr(getAwsEc2ListMultiResponseReader(), "/DescribeVolumesResponse/volumeSet/item")
	assert.NilError(t, err)
	assert.Assert(t, m != nil)
	assert.Assert(t, m[0]["volumeId"] == "vol-001ebed16c2567746")
	assert.Assert(t, m[1]["volumeId"] == "vol-024a257300c66ed56")
}

func TestAwareListVolumesMulti(t *testing.T) {

	fr := getFileRoot(t)

	l := openapi3.NewLoader()
	svc, err := l.LoadFromFile(filepath.Join(fr, "aws", "v0.1.0", "services", "ec2.yaml"))
	assert.NilError(t, err)
	assert.Assert(t, svc != nil)

	m, err := GetSubObjArr(getAwsEc2ListMultiResponseReader(), "/DescribeVolumesResponse/volumeSet/item")
	assert.NilError(t, err)
	assert.Assert(t, m != nil)
	assert.Assert(t, m[0]["volumeId"] == "vol-001ebed16c2567746")
	assert.Assert(t, m[1]["volumeId"] == "vol-024a257300c66ed56")
}

var (
	awsEc2ListResponseSingle string = `
	<?xml version="1.0" encoding="UTF-8"?>
	<DescribeVolumesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
			<requestId>7f833cd4-1440-4ce9-be7d-808439ace59a</requestId>
			<volumeSet>
					<item>
							<volumeId>vol-001ebed16c2567746</volumeId>
							<size>10</size>
							<snapshotId/>
							<availabilityZone>ap-southeast-1a</availabilityZone>
							<status>available</status>
							<createTime>2022-05-02T23:09:30.171Z</createTime>
							<attachmentSet/>
							<volumeType>gp2</volumeType>
							<iops>100</iops>
							<encrypted>false</encrypted>
							<multiAttachEnabled>false</multiAttachEnabled>
					</item>
			</volumeSet>
	</DescribeVolumesResponse>
	`

	awsEc2ListResponseMulti string = `
	<?xml version="1.0" encoding="UTF-8"?>
	<DescribeVolumesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
			<requestId>6b5e0474-042b-45d6-adac-04b0aff9ab10</requestId>
			<volumeSet>
					<item>
							<volumeId>vol-001ebed16c2567746</volumeId>
							<size>10</size>
							<snapshotId/>
							<availabilityZone>ap-southeast-1a</availabilityZone>
							<status>available</status>
							<createTime>2022-05-02T23:09:30.171Z</createTime>
							<attachmentSet/>
							<volumeType>gp2</volumeType>
							<iops>100</iops>
							<encrypted>false</encrypted>
							<multiAttachEnabled>false</multiAttachEnabled>
					</item>
					<item>
							<volumeId>vol-024a257300c66ed56</volumeId>
							<size>8</size>
							<snapshotId/>
							<availabilityZone>ap-southeast-1a</availabilityZone>
							<status>available</status>
							<createTime>2022-05-11T04:45:40.627Z</createTime>
							<attachmentSet/>
							<volumeType>gp2</volumeType>
							<iops>100</iops>
							<encrypted>false</encrypted>
							<multiAttachEnabled>false</multiAttachEnabled>
					</item>
			</volumeSet>
	</DescribeVolumesResponse>
	`
)

func getAwsEc2ListSingleResponseReader() io.ReadCloser {
	return io.NopCloser(bytes.NewBufferString(awsEc2ListResponseSingle))
}

func getAwsEc2ListMultiResponseReader() io.ReadCloser {
	return io.NopCloser(bytes.NewBufferString(awsEc2ListResponseMulti))
}
