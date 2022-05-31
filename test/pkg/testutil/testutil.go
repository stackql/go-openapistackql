package testutil

import (
	"bytes"
	"io"
	"os"
	"path"

	"github.com/stackql/go-openapistackql/pkg/fileutil"
)

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
							<createTime>2020-05-02T23:09:30.171Z</createTime>
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
							<createTime>2020-05-02T23:09:30.171Z</createTime>
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

func GetAwsEc2ListSingleResponseReader() io.ReadCloser {
	return io.NopCloser(bytes.NewBufferString(awsEc2ListResponseSingle))
}

func GetAwsEc2ListMultiResponseReader() io.ReadCloser {
	return io.NopCloser(bytes.NewBufferString(awsEc2ListResponseMulti))
}

func GetK8SNodesListMultiResponseReader() (io.ReadCloser, error) {
	f, err := fileutil.GetFilePathFromRepositoryRoot(path.Join("test", "input", "k8s-nodes.json"))
	if err != nil {
		return nil, err
	}
	return os.Open(f)
}
