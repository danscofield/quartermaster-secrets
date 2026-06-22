package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(`
server:
  addr: ":9443"
  tls:
    cert_file: cert.pem
    key_file: key.pem
aws:
  region: eu-west-1
dynamodb:
  table: my-secrets
secrets_manager:
  prefix: app-secrets
oidc:
  issuer: https://idp.example.com
  audience: qm-secrets
  insecure_skip_tls_verify: true
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cert.pem"), []byte("cert"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "key.pem"), []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Addr != ":9443" {
		t.Fatalf("addr %q", cfg.Addr)
	}
	if !cfg.TLSEnabled() {
		t.Fatal("expected tls enabled")
	}
	if cfg.AWSRegion != "eu-west-1" {
		t.Fatalf("region %q", cfg.AWSRegion)
	}
	if cfg.DynamoDBTable != "my-secrets" {
		t.Fatalf("table %q", cfg.DynamoDBTable)
	}
	if cfg.ASMSecretPrefix != "app-secrets" {
		t.Fatalf("prefix %q", cfg.ASMSecretPrefix)
	}
	if cfg.OIDCIssuer != "https://idp.example.com" {
		t.Fatalf("issuer %q", cfg.OIDCIssuer)
	}
	if cfg.OIDCAudience != "qm-secrets" {
		t.Fatalf("audience %q", cfg.OIDCAudience)
	}
	if !cfg.OIDCInsecureSkipTLSVerify {
		t.Fatal("expected insecure skip tls")
	}
}

func TestLoadEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(`
dynamodb:
  table: from-file
oidc:
  issuer: https://idp.example.com
`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DYNAMODB_TABLE", "from-env")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DynamoDBTable != "from-env" {
		t.Fatalf("table %q, want from-env", cfg.DynamoDBTable)
	}
}

func TestLoadEnvOnly(t *testing.T) {
	t.Setenv("DYNAMODB_TABLE", "env-table")
	t.Setenv("OIDC_ISSUER", "https://idp.example.com")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DynamoDBTable != "env-table" {
		t.Fatalf("table %q", cfg.DynamoDBTable)
	}
	if cfg.Addr != ":8080" {
		t.Fatalf("addr %q", cfg.Addr)
	}
}

func TestLoadBootstrapFromFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(`
aws:
  region: ap-southeast-2
dynamodb:
  table: bootstrap-table
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadBootstrap(path)
	if err != nil {
		t.Fatalf("LoadBootstrap: %v", err)
	}
	if cfg.DynamoDBTable != "bootstrap-table" {
		t.Fatalf("table %q", cfg.DynamoDBTable)
	}
	if cfg.AWSRegion != "ap-southeast-2" {
		t.Fatalf("region %q", cfg.AWSRegion)
	}
}

func TestValidatePartialTLS(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(`
dynamodb:
  table: t
oidc:
  issuer: https://idp.example.com
server:
  tls:
    cert_file: cert.pem
`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for partial tls config")
	}
}
