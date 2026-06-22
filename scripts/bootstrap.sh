#!/usr/bin/env bash
set -euo pipefail

# Bootstrap AWS resources for qm-secrets.
#
# Reads config from -config / QM_SECRETS_CONFIG / config.yaml (see README).
# Environment variables override file values:
#   DYNAMODB_TABLE  Table name
#   AWS_REGION      AWS region
#
# Uses the default AWS credential chain (env vars, ~/.aws/credentials, IAM role).

cd "$(dirname "$0")/.."
go run ./cmd/bootstrap "$@"
