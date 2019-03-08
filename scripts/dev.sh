#!/usr/bin/env bash
set -e
set -x

#
# Helper script for local development. Automatically builds and registers the
# plugin. Requires `vault` is installed and available on $PATH.
#

# Get the right dir
DIR="$(cd "$(dirname "$(readlink "$0")")" && pwd)"

echo "==> Starting dev"

echo "--> Scratch dir"
SCRATCH="$DIR/tmp"
mkdir -p "$SCRATCH/plugins"

echo "--> IPFS node"
ipfs daemon &
sleep 2
IPFS_PID=$1

echo "--> Vault server"
echo "    Writing config"
tee "$SCRATCH/vault.hcl" > /dev/null <<EOF
plugin_directory = "$SCRATCH/plugins"
ui = true

listener "tcp" {
  max_request_duration = "30s" # duration allowed before Vault cancels request
  max_request_size = 0         # disable request size limit (bad)
}
EOF

echo "    Envvars"
export VAULT_DEV_ROOT_TOKEN_ID="root"
export VAULT_ADDR="http://127.0.0.1:8200"

echo "    Starting"
vault server \
  -dev \
  -config="$SCRATCH/vault.hcl" \
  &
sleep 2
VAULT_PID=$!

function cleanup {
  echo ""
  echo "==> Cleaning up"
  kill -INT "$VAULT_PID"
  rm -rf "$SCRATCH"
}
# trap cleanup EXIT

echo "--> Authing"
vault login root &>/dev/null

echo "--> Building"
go build -o "$SCRATCH/plugins/vault-ipfs" ./cmd/vault-plugin-ipfs
SHASUM=$(shasum -a 256 "$SCRATCH/plugins/vault-ipfs" | cut -d " " -f1)

echo "--> Registering plugin"
vault write sys/plugins/catalog/secret/ipfs \
  sha_256="$SHASUM" \
  command="vault-ipfs"

echo "--> Mouting plugin"
vault secrets enable -path=ipfs -plugin-name=ipfs plugin

echo "--> Reading out"
vault read -field=data ipfs/dag/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/readme \
  | base64 -D

echo "==> Ready!"
wait $!
