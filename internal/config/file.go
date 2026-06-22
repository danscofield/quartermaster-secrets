package config

// File is the on-disk configuration schema (YAML).
type File struct {
	Server          ServerFile          `yaml:"server"`
	AWS             AWSFile             `yaml:"aws"`
	DynamoDB        DynamoDBFile        `yaml:"dynamodb"`
	SecretsManager  SecretsManagerFile  `yaml:"secrets_manager"`
	OIDC            OIDCFile            `yaml:"oidc"`
}

type ServerFile struct {
	Addr string   `yaml:"addr"`
	TLS  TLSFile  `yaml:"tls"`
}

type TLSFile struct {
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type AWSFile struct {
	Region string `yaml:"region"`
}

type DynamoDBFile struct {
	Table string `yaml:"table"`
}

type SecretsManagerFile struct {
	Prefix string `yaml:"prefix"`
}

type OIDCFile struct {
	Issuer                string `yaml:"issuer"`
	Audience              string `yaml:"audience"`
	InsecureSkipTLSVerify bool   `yaml:"insecure_skip_tls_verify"`
}
