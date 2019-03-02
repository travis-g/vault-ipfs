package ipfs

import (
	"context"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	"github.com/pkg/errors"
)

// Use local node. The public gateway is slow.
const ipfsAddr = "127.0.0.1:5001"

// Factory creates a new usable instance of this secrets engine.
func Factory(ctx context.Context, c *logical.BackendConfig) (logical.Backend, error) {
	b := New(c)
	if err := b.Setup(ctx, c); err != nil {
		return nil, errors.Wrap(err, "failed to create factory")
	}
	return b, nil
}

type Backend struct {
	*framework.Backend
}

func New(c *logical.BackendConfig) *Backend {
	var b Backend
	b.Backend = &framework.Backend{
		Help: ``,
		Paths: framework.PathAppend(
			b.objectPaths(),
			b.statusPaths(),
			b.dagPaths(),
		),
		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{},
		},
		Secrets:     []*framework.Secret{},
		BackendType: logical.TypeLogical,
	}
	return &b
}
