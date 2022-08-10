package querytranspose

const (
	AWSCanonicalAlgorithm    string = "AWSCanonical"
	AWSCloudControlAlgorithm string = "AWSCloudControl"
)

type QueryTransposer interface {
	Transpose() (map[string]string, error)
}

func NewQueryTransposer(algorithm string, rawInput []byte, baseKey string) QueryTransposer {
	switch algorithm {
	case AWSCanonicalAlgorithm:
		return newAWSCanonicalQueryTransposer(rawInput, baseKey)
	case AWSCloudControlAlgorithm:
		return newAWSCloudControlQueryTransposer(rawInput, baseKey)
	default:
		return newAWSCloudControlQueryTransposer(rawInput, baseKey)
	}
}
