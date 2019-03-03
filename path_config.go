package ipfs

import (
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

var configFields = map[string]*framework.FieldSchema{
	"address": &framework.FieldSchema{
		Type:        framework.TypeString,
		Description: "[Required] address of the IPFS API to use",
	},
	"format": &framework.FieldSchema{
		Type:        framework.TypeString,
		Default:     "dag-pb",
		Description: "format that the object will be added as",
	},
	"input-enc": &framework.FieldSchema{
		Type:        framework.TypeString,
		Default:     "json",
		Description: "format that the input object will be",
	},
	"pin": &framework.FieldSchema{
		Type:        framework.TypeBool,
		Default:     true,
		Description: "pin the object when added",
	},
	"hash": &framework.FieldSchema{
		Type:        framework.TypeString,
		Default:     "sha2-256",
		Description: "multihash hashing algorithm to use",
	},
}

func (b *backend) configPaths() []*framework.Path {
	return []*framework.Path{
		&framework.Path{
			Pattern:      "config",
			HelpSynopsis: "Configures backend settings applied to all data",
			Fields:       configFields,
			Callbacks:    map[logical.Operation]framework.OperationFunc{
				// logical.UpdateOperation: ,
				// logical.ReadOperation: ,
			},
		},
	}
}
