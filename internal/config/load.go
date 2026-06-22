package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the resolved runtime configuration.
type Config struct {
	Addr                      string
	DynamoDBTable             string
	ASMSecretPrefix           string
	OIDCIssuer                string
	OIDCAudience              string
	OIDCInsecureSkipTLSVerify bool
	AWSRegion                 string
	TLSCertFile               string
	TLSKeyFile                string
}

// BootstrapConfig holds settings for infrastructure bootstrap commands.
type BootstrapConfig struct {
	DynamoDBTable string
	AWSRegion     string
}

// TLSEnabled reports whether the server should listen with TLS.
func (c Config) TLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != ""
}

// Load reads configuration from an optional YAML file, then applies env overrides.
// path may be empty; see ResolveConfigPath for search order.
func Load(path string) (Config, error) {
	file, err := readFile(path)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Addr:                      ":8080",
		ASMSecretPrefix:           "qm-secrets",
		AWSRegion:                 "us-east-1",
		OIDCInsecureSkipTLSVerify: false,
	}
	applyFile(&cfg, file)
	applyEnv(&cfg)

	if err := validateServer(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// LoadBootstrap reads bootstrap configuration from file and env.
func LoadBootstrap(path string) (BootstrapConfig, error) {
	file, err := readFile(path)
	if err != nil {
		return BootstrapConfig{}, err
	}

	cfg := BootstrapConfig{
		DynamoDBTable: "qm-secrets",
		AWSRegion:     "us-east-1",
	}
	if file.DynamoDB.Table != "" {
		cfg.DynamoDBTable = file.DynamoDB.Table
	}
	if file.AWS.Region != "" {
		cfg.AWSRegion = file.AWS.Region
	}
	if v := os.Getenv("DYNAMODB_TABLE"); v != "" {
		cfg.DynamoDBTable = v
	}
	if v := os.Getenv("AWS_REGION"); v != "" {
		cfg.AWSRegion = v
	}
	if cfg.DynamoDBTable == "" {
		return cfg, fmt.Errorf("dynamodb.table is required")
	}
	return cfg, nil
}

// ResolveConfigPath returns the config file path from flag, env, or well-known locations.
func ResolveConfigPath(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if v := os.Getenv("QM_SECRETS_CONFIG"); v != "" {
		return v
	}
	for _, p := range []string{"config.yaml", "qm-secrets.yaml", "/etc/qm-secrets/config.yaml"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func readFile(path string) (File, error) {
	path = ResolveConfigPath(path)
	if path == "" {
		return File{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var file File
	if err := yaml.Unmarshal(data, &file); err != nil {
		return File{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	return file, nil
}

func applyFile(cfg *Config, file File) {
	if file.Server.Addr != "" {
		cfg.Addr = file.Server.Addr
	}
	if file.Server.TLS.CertFile != "" {
		cfg.TLSCertFile = file.Server.TLS.CertFile
	}
	if file.Server.TLS.KeyFile != "" {
		cfg.TLSKeyFile = file.Server.TLS.KeyFile
	}
	if file.AWS.Region != "" {
		cfg.AWSRegion = file.AWS.Region
	}
	if file.DynamoDB.Table != "" {
		cfg.DynamoDBTable = file.DynamoDB.Table
	}
	if file.SecretsManager.Prefix != "" {
		cfg.ASMSecretPrefix = file.SecretsManager.Prefix
	}
	if file.OIDC.Issuer != "" {
		cfg.OIDCIssuer = file.OIDC.Issuer
	}
	if file.OIDC.Audience != "" {
		cfg.OIDCAudience = file.OIDC.Audience
	}
	if file.OIDC.InsecureSkipTLSVerify {
		cfg.OIDCInsecureSkipTLSVerify = true
	}
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("ADDR"); v != "" {
		cfg.Addr = v
	}
	if v := os.Getenv("TLS_CERT_FILE"); v != "" {
		cfg.TLSCertFile = v
	}
	if v := os.Getenv("TLS_KEY_FILE"); v != "" {
		cfg.TLSKeyFile = v
	}
	if v := os.Getenv("AWS_REGION"); v != "" {
		cfg.AWSRegion = v
	}
	if v := os.Getenv("DYNAMODB_TABLE"); v != "" {
		cfg.DynamoDBTable = v
	}
	if v := os.Getenv("ASM_SECRET_PREFIX"); v != "" {
		cfg.ASMSecretPrefix = v
	}
	if v := os.Getenv("OIDC_ISSUER"); v != "" {
		cfg.OIDCIssuer = v
	}
	if v := os.Getenv("OIDC_AUDIENCE"); v != "" {
		cfg.OIDCAudience = v
	}
	if envBool("OIDC_INSECURE_SKIP_TLS_VERIFY") {
		cfg.OIDCInsecureSkipTLSVerify = true
	}
}

func validateServer(cfg Config) error {
	if cfg.DynamoDBTable == "" {
		return fmt.Errorf("dynamodb.table is required")
	}
	if cfg.OIDCIssuer == "" {
		return fmt.Errorf("oidc.issuer is required")
	}
	if (cfg.TLSCertFile == "") != (cfg.TLSKeyFile == "") {
		return fmt.Errorf("server.tls.cert_file and server.tls.key_file must both be set")
	}
	if cfg.TLSCertFile != "" {
		if err := fileExists(cfg.TLSCertFile); err != nil {
			return fmt.Errorf("server.tls.cert_file: %w", err)
		}
		if err := fileExists(cfg.TLSKeyFile); err != nil {
			return fmt.Errorf("server.tls.key_file: %w", err)
		}
	}
	return nil
}

func fileExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%q is a directory", path)
	}
	return nil
}

func envBool(key string) bool {
	switch strings.ToLower(os.Getenv(key)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
