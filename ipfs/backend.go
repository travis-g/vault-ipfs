package ipfs

import (
	"context"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	"github.com/pkg/errors"
)

const ipfsAddr = "127.0.0.1:5001"

// Factory creates a new usable instance of this secrets engine.
func Factory(ctx context.Context, c *logical.BackendConfig) (logical.Backend, error) {
	b := Backend(c)
	if err := b.Setup(ctx, c); err != nil {
		return nil, errors.Wrap(err, "failed to create factory")
	}
	return b, nil
}

type IPFSBackend struct {
	*framework.Backend
}

func Backend(c *logical.BackendConfig) *IPFSBackend {
	var b IPFSBackend
	b.Backend = &framework.Backend{
		Help:  ``,
		Paths: framework.PathAppend(objectPaths(&b)),
		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{},
		},
		Secrets:     []*framework.Secret{},
		BackendType: logical.TypeLogical,
	}
	return &b
}
