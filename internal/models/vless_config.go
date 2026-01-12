package models

type VlessConfig struct {
	URL        string
	IP         string
	Port       string
	SNI        string
	TestResult int
}

type VlessURL struct {
	UUID        string
	Server      string
	Port        int
	Security    string
	SNI         string
	Fingerprint string
	PublicKey   string
	SID         string
	SPX         string
	Type        string
	Flow        string

	Path        string
	Host        string
	ServiceName string
	HeaderType  string
}
