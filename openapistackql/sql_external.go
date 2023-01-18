package openapistackql

type SQLExternalConnection struct {
	ConnectionName string                      `json:"alias" yaml:"alias"`
	Tables         map[string]SQLExternalTable `json:"tables" yaml:"tables"`
}

type SQLExternalTable struct {
	CatalogName string              `json:"catalogName" yaml:"catalogName"`
	SchemaName  string              `json:"schemaName" yaml:"schemaName"`
	Name        string              `json:"name" yaml:"name"`
	Columns     []SQLExternalColumn `json:"columns" yaml:"columns"`
}

type SQLExternalColumn struct {
	Name      string `json:"name" yaml:"name"`
	Type      string `json:"type" yaml:"type"`
	Oid       uint32 `json:"oid" yaml:"oid"`
	Width     int    `json:"width" yaml:"width"`
	Precision int    `json:"precision" yaml:"precision"`
	// OrdinalPosition int    `json:"ordinalPosition" yaml:"ordinalPosition"`
}
