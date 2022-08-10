package querytranspose

type AWSCloudControlQueryTransposer struct {
	baseKey  string
	rawInput []byte
}

func newAWSCloudControlQueryTransposer(rawInput []byte, baseKey string) QueryTransposer {
	return &AWSCloudControlQueryTransposer{
		rawInput: rawInput,
		baseKey:  baseKey,
	}
}

func (um *AWSCloudControlQueryTransposer) Transpose() (map[string]string, error) {
	return map[string]string{
		um.baseKey: string(um.rawInput),
	}, nil
}
