#!/usr/bin/env bash
# Converts a PEM private key file into the single-line \n-escaped format
# required for the GITHUB_APP_PRIVATE_KEY environment variable.
#
# Usage:
#   ./scripts/pem-to-env.sh path/to/private-key.pem
#
# Output (copy the printed line into your .env):
#   GITHUB_APP_PRIVATE_KEY=-----BEGIN RSA PRIVATE KEY-----\nMII...\n-----END RSA PRIVATE KEY-----

set -euo pipefail

if [ $# -ne 1 ]; then
  echo "Usage: $0 <path-to-pem-file>" >&2
  exit 1
fi

PEM_FILE="$1"

if [ ! -f "$PEM_FILE" ]; then
  echo "Error: file not found: $PEM_FILE" >&2
  exit 1
fi

printf 'GITHUB_APP_PRIVATE_KEY='
awk 'NF { printf "%s\\n", $0 }' "$PEM_FILE"
printf '\n'
