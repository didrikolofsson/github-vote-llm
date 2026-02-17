#!/bin/sh
set -e

# If GITHUB_APP_PRIVATE_KEY is set, write it to a .pem file
# so the app can reference it via private_key_path in config.yaml.
if [ -n "$GITHUB_APP_PRIVATE_KEY" ]; then
  mkdir -p /etc/vote-llm
  printf '%s\n' "$GITHUB_APP_PRIVATE_KEY" > /etc/vote-llm/app-key.pem
  chmod 600 /etc/vote-llm/app-key.pem
fi

exec vote-llm "$@"
