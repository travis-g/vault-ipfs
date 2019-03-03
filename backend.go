package ipfs

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/vault/helper/keysutil"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	"github.com/pkg/errors"
)

// const ipfsAddr = "https://ipfs.infura.io:5001"
const ipfsAddr = "http://127.0.0.1:5001"

var ipfsClient = &http.Client{
	Timeout: time.Second * 10,
}

// Factory returns an IPFS backend that satisfies the logical.Backend interface
func Factory(ctx context.Context, c *logical.BackendConfig) (logical.Backend, error) {
	b := Backend(c)
	if err := b.Setup(ctx, c); err != nil {
		return nil, errors.Wrap(err, "failed to create factory")
	}
	return b, nil
}

type backend struct {
	*framework.Backend
	lm *keysutil.LockManager
}

func Backend(c *logical.BackendConfig) *backend {
	var b backend
	b.Backend = &framework.Backend{
		Help: strings.TrimSpace(backendIPFSHelp),
		Paths: framework.PathAppend(
			b.objectPaths(),
			b.statusPaths(),
			b.dagPaths(),
		),
		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{},
		},
		Secrets: []*framework.Secret{
			secretKey(&b),
		},
		BackendType: logical.TypeLogical,
	}

	b.lm = keysutil.NewLockManager(c.System.CachingDisabled())

	return &b
}

// TODO
func (b *backend) client(ctx context.Context, s logical.Storage) (*http.Client, error) {
	return nil, nil
}

const backendIPFSHelp = `
The IPFS backend generates keys and provides an abstraction layer over IPFS.
`
