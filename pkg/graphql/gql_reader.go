package graphql

type GQLReader interface {
	Read() ([]map[string]interface{}, error)
}
