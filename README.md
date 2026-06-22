# qm-secrets

A small REST API for secrets management backed by **Amazon DynamoDB** (metadata + billet ACLs) and **AWS Secrets Manager** (secret values).

## Architecture

```
Client (JWT with billets)
        │
        ▼
   REST API (Go)
        │
        ├── DynamoDB  ── name, owners, retrievers, asm_secret_arn, last_updated
        └── ASM       ── { "value1": "...", "value2": "..." }
```

- **Owners** can update billet metadata and set `value1` / `value2` (green/blue).
- **Retrievers** can list and retrieve secrets they have access to.
- On create or metadata update, at least one owner billet must overlap with the caller's JWT billets.
- Value updates set `last_updated`; clients poll for changes without fetching every secret from ASM.

## API

All endpoints except `/health` require `Authorization: Bearer <JWT>` issued by your OIDC IdP.

The token is validated via the issuer's JWKS (discovered from `/.well-known/openid-configuration`). When `OIDC_AUDIENCE` is set, the token's `aud` claim must match.

JWT claim shape:

```json
{
  "iss": "https://idp.example.com",
  "aud": "qm-secrets",
  "billets": ["team-a", "team-b"]
}
```

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness check |
| GET | `/docs` | Swagger UI (redirects to `/docs/`) |
| GET | `/openapi.yaml` | OpenAPI 3 spec |
| POST | `/secrets` | Create a secret |
| GET | `/secrets` | List secrets visible to caller |
| GET | `/secrets/{name}` | Retrieve secret (includes values from ASM) |
| PUT | `/secrets/{name}` | Update metadata and/or values |
| DELETE | `/secrets/{name}` | Delete secret (owners only) |
| POST | `/secrets/poll` | Return secrets updated since client's last known `last_updated` |

### Create secret

```json
POST /secrets
{
  "name": "my-app/config",
  "owners": ["team-a"],
  "retrievers": ["team-b", "team-c"],
  "value1": "green-value",
  "value2": "blue-value"
}
```

### Poll for updates

```json
POST /secrets/poll
{
  "secrets": [
    { "name": "my-app/config", "last_updated": "2026-06-21T12:00:00Z" }
  ]
}
```

Response:

```json
{
  "updated": [
    {
      "name": "my-app/config",
      "owners": ["team-a"],
      "retrievers": ["team-b"],
      "last_updated": "2026-06-21T14:30:00Z",
      "can_update": true
    }
  ]
}
```

Clients should call `GET /secrets/{name}` for each entry in `updated` to fetch new values from ASM.

## Bootstrap

Create the DynamoDB table (idempotent — safe to re-run):

```bash
./scripts/bootstrap.sh
# or: go run ./cmd/bootstrap -config config.yaml
```

The table uses `name` (String) as the partition key and on-demand billing.
Item attributes (`owners`, `retrievers`, `asm_secret_arn`, `last_updated`) are
schemaless and created by the application at write time.

## DynamoDB table

Create a table with partition key `name` (String):

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | S | Secret name (PK) |
| `owners` | L | Owner billets |
| `retrievers` | L | Retriever billets |
| `asm_secret_arn` | S | ARN of the ASM secret |
| `last_updated` | S | RFC3339 timestamp |

You can also create the table manually:

```bash
aws dynamodb create-table \
  --table-name qm-secrets \
  --attribute-definitions AttributeName=name,AttributeType=S \
  --key-schema AttributeName=name,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST
```

## Configuration

Copy `config.example.yaml` to `config.yaml` and edit. The server and bootstrap
commands load it automatically from the current directory (or pass `-config`).

```bash
cp config.example.yaml config.yaml
```

Search order: `-config` flag → `QM_SECRETS_CONFIG` env → `config.yaml` →
`qm-secrets.yaml` → `/etc/qm-secrets/config.yaml`. If no file is found, env vars
alone still work.

### config.yaml

```yaml
server:
  addr: ":8080"
  tls:
    cert_file: /etc/qm-secrets/tls.crt
    key_file: /etc/qm-secrets/tls.key

aws:
  region: us-east-1

dynamodb:
  table: qm-secrets

secrets_manager:
  prefix: qm-secrets

oidc:
  issuer: https://idp.example.com
  audience: qm-secrets
  insecure_skip_tls_verify: false   # dev only
```

### Environment overrides

Any file value can be overridden at runtime:

| Variable | Config key | Description |
|----------|------------|-------------|
| `QM_SECRETS_CONFIG` | — | Path to YAML config file |
| `ADDR` | `server.addr` | HTTP listen address |
| `TLS_CERT_FILE` | `server.tls.cert_file` | TLS certificate path |
| `TLS_KEY_FILE` | `server.tls.key_file` | TLS private key path |
| `AWS_REGION` | `aws.region` | AWS region |
| `DYNAMODB_TABLE` | `dynamodb.table` | DynamoDB table name |
| `ASM_SECRET_PREFIX` | `secrets_manager.prefix` | Prefix for ASM secret names |
| `OIDC_ISSUER` | `oidc.issuer` | OIDC issuer URL |
| `OIDC_AUDIENCE` | `oidc.audience` | Expected `aud` claim |
| `OIDC_INSECURE_SKIP_TLS_VERIFY` | `oidc.insecure_skip_tls_verify` | Skip TLS for OIDC/JWKS fetches (**dev only**) |

Both `server.tls.cert_file` and `server.tls.key_file` must be set to enable HTTPS.

IAM permissions needed:

- DynamoDB: `GetItem`, `PutItem`, `UpdateItem`, `DeleteItem`, `Scan` on the table
- Secrets Manager: `CreateSecret`, `GetSecretValue`, `PutSecretValue`, `DeleteSecret`

## Run

```bash
cp config.example.yaml config.yaml
# edit config.yaml

go run ./cmd/server
# or: go run ./cmd/server -config /path/to/config.yaml
```

For a local IdP with a self-signed certificate, set `oidc.insecure_skip_tls_verify: true`
in config (or `OIDC_INSECURE_SKIP_TLS_VERIFY=true`).

## Build

```bash
go build -o bin/qm-secrets ./cmd/server
go test ./...
```

## API documentation

Interactive docs are served at [`/docs`](http://localhost:8080/docs/) (Swagger UI).
The OpenAPI 3 spec is available at [`/openapi.yaml`](http://localhost:8080/openapi.yaml).
The spec source lives in `internal/docs/openapi.yaml`.
