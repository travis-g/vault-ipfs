package ipfs

import "github.com/hashicorp/vault/logical/framework"

const SecretIPFSKeyType = "secret_ipfs_key_type"

func secretKey(b *backend) *framework.Secret {
	return &framework.Secret{
		Type: SecretIPFSKeyType,
		Fields: map[string]*framework.FieldSchema{
			"key": {
				Type:        framework.TypeString,
				Description: "Encryption key",
			},
		},
	}
}
