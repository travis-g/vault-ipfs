// `reader` is an example policy designed to read managed objects out of IPFS
// through Vault and decrypt them.

path "ipfs/data/*" {
  capabilities = ["read"]
}

path "ipfs/metadata/*" {
  capabilities = ["list"]
}

// Decrypt data with the transit backend.
path "transit/decrypt/local" {
  capabilities = ["update"]
}
