// `importer` is an example policy designed to import objects into Vault through
// plugin's API. The policy also grants the ability to proxy reads of the entire
// IPFS Merkle forest.

// Read any node of the IPFS Merkle forest.
path "ipfs/object/*" {
  capabilities = ["list", "read"]
}

// POST objects and data to Vault to be encrypted and managed.
path "ipfs/data" {
  capabilities = ["create"]
}

// Enumerate objects under management.
path "ipfs/data/*" {
  capabilities = ["list"]
}
