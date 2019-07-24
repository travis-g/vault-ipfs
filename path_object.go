package ipfs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	ipfs "github.com/ipfs/go-ipfs-api"
)

var objectFields = map[string]*framework.FieldSchema{
	"key": {
		Type:        framework.TypeString,
		Description: "DAG node to pull and serialize from IPFS",
	},
	"link": {
		Type:        framework.TypeString,
		Description: "optional link of DAG node to pull",
	},
}

func (b *backend) objectPaths() []*framework.Path {
	return []*framework.Path{
		// The order of these paths matters: more specific ones need to be near
		// the top, so that path matching does not short-circuit.
		{
			Pattern:      "object/" + framework.GenericNameRegex("key") + framework.OptionalParamRegex("link") + "/",
			HelpSynopsis: "Return an IPFS object's links",
			Fields:       objectFields,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ListOperation: b.pathObjectLinks,
			},
		},
		{
			Pattern:      "object/" + framework.GenericNameRegex("key") + framework.OptionalParamRegex("link"),
			HelpSynopsis: "Return an IPFS DAG node",
			Fields:       objectFields,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation: b.pathObjectGet,
			},
		},
		{
			Pattern:      "object/" + framework.GenericNameRegex("key") + "/",
			HelpSynopsis: "Return a list of an IPFS object's links",
			Fields:       objectFields,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ListOperation: b.pathObjectLinks,
			},
		},
	}
}

/*
pathObjectGet returns an IpfsObject DAG node as returned by the network.

- Use encoding/json to decode strings, ex.
  https://ipfs.infura.io:5001/api/v0/object/get?arg=QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG
*/
func (b *backend) pathObjectGet(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	if err := validateFields(req, d); err != nil {
		return nil, logical.CodedError(http.StatusUnprocessableEntity, err.Error())
	}

	sh := ipfs.NewShell(ipfsAddr)

	key := d.Get("key").(string)
	link := d.Get("link").(string)
	if link != "" {
		key = key + "/" + link
	}

	// Get object from IPFS
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
func (b *backend) pathObjectLinks(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	if err := validateFields(req, d); err != nil {
		return nil, logical.CodedError(http.StatusUnprocessableEntity, err.Error())
	}

	sh := ipfs.NewShell(ipfsAddr)

	key := d.Get("key").(string)
	link := d.Get("link").(string)
	if link != "" {
		key = key + "/" + link
	}

	// Get object from IPFS
	object, err := sh.ObjectGet(key)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	// Pull just the links out of the DAG
	hashes := make([]string, 0, len(object.Links))
	for _, link := range object.Links {
		hashes = append(hashes, link.Hash)
	}

	return logical.ListResponse(hashes), nil
}
