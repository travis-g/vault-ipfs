package ipfs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	shell "github.com/ipfs/go-ipfs-api"
)

var objectFields = map[string]*framework.FieldSchema{
	"key": &framework.FieldSchema{
		Type:        framework.TypeString,
		Description: "DAG node to pull and serialize from IPFS",
	},
	"link": &framework.FieldSchema{
		Type:        framework.TypeString,
		Description: "optional link of DAG node to pull",
	},
}

func objectPaths(b *IPFSBackend) []*framework.Path {
	return []*framework.Path{
		&framework.Path{
			Pattern:      "object/" + framework.GenericNameRegex("key") + framework.OptionalParamRegex("link"),
			HelpSynopsis: "Return an IPFS DAG node",
			Fields:       objectFields,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation: b.pathObjectGet,
			},
		},
		&framework.Path{
			Pattern:      "object/" + framework.GenericNameRegex("key") + "/",
			HelpSynopsis: "Return an IPFS object's links",
			Fields:       objectFields,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ListOperation: b.pathObjectLinks,
			},
		},
	}
}

// pathObjectGet returns an IpfsObject DAG node as returned by the network.
func (b *IPFSBackend) pathObjectGet(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	if err := validateFields(req, d); err != nil {
		return nil, logical.CodedError(http.StatusUnprocessableEntity, err.Error())
	}

	sh := shell.NewShell(ipfsAddr)

	key := d.Get("key").(string)
	link := d.Get("link").(string)
	if link != "" {
		key = strings.TrimRight(key+"/"+link, "/")
	}

	object, err := sh.ObjectGet(key)
	if err != nil {
		return nil, logical.CodedError(http.StatusNotFound, err.Error())
	}

	// base64 encode payload and update in-place
	object.Data = base64.StdEncoding.EncodeToString([]byte(object.Data))

	var data map[string]interface{}
	jsonBytes, err := json.Marshal(object)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}
	json.Unmarshal(jsonBytes, &data)

	return &logical.Response{
		Data: data,
	}, nil
}

// pathObjectLinks returns a list of hashes linked to by an IpfsObject.
func (b *IPFSBackend) pathObjectLinks(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	if err := validateFields(req, d); err != nil {
		return nil, logical.CodedError(http.StatusUnprocessableEntity, err.Error())
	}

	sh := shell.NewShell(ipfsAddr)

	key := d.Get("key").(string)
	link := d.Get("link").(string)
	if link != "" {
		key = strings.TrimRight(key+"/"+link, "/")
	}

	object, err := sh.ObjectGet(key)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	hashes := make([]string, 0, len(object.Links))
	for _, link := range object.Links {
		hashes = append(hashes, link.Hash)
	}

	return logical.ListResponse(hashes), nil
}
