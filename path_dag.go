package ipfs

import (
	"context"
	"net/http"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	ipfs "github.com/ipfs/go-ipfs-api"
)

var dagFields = map[string]*framework.FieldSchema{
	"ref": &framework.FieldSchema{
		Type:        framework.TypeString,
		Description: "DAG node to pull and serialize from IPFS",
	},
	"link": &framework.FieldSchema{
		Type:        framework.TypeString,
		Description: "optional link of DAG node to pull",
	},
}

func (b *Backend) dagPaths() []*framework.Path {
	return []*framework.Path{
		// The order of these paths matters: more specific ones need to be near
		// the top, so that path matching does not short-circuit.
		&framework.Path{
			Pattern:      "dag/" + framework.GenericNameRegex("ref") + framework.OptionalParamRegex("link") + "/",
			HelpSynopsis: "Return an IPLS DAG's Links",
			Fields:       dagFields,
			Callbacks:    map[logical.Operation]framework.OperationFunc{
				// logical.ListOperation: b.pathDAGLinks,
			},
		},
		&framework.Path{
			Pattern:      "dag/" + framework.GenericNameRegex("ref") + framework.OptionalParamRegex("link"),
			HelpSynopsis: "Return a DAG node",
			Fields:       dagFields,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation: b.pathDAGGet,
			},
		},
		&framework.Path{
			Pattern:      "dag/" + framework.GenericNameRegex("ref") + "/",
			HelpSynopsis: "Return a list of a DAG's Links",
			Fields:       dagFields,
			Callbacks:    map[logical.Operation]framework.OperationFunc{
				// logical.ListOperation: b.pathDAGLinks,
			},
		},
	}
}

// pathDAGGet returns a DAG node as returned by the network.
func (b *Backend) pathDAGGet(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	if err := validateFields(req, d); err != nil {
		return nil, logical.CodedError(http.StatusUnprocessableEntity, err.Error())
	}

	sh := ipfs.NewShell(ipfsAddr)

	key := d.Get("ref").(string)
	link := d.Get("link").(string)
	if link != "" {
		key = key + "/" + link
	}

	// Get object from IPFS
	var dag string
	err := sh.DagGet(key, dag)
	if err != nil {
		return nil, logical.CodedError(http.StatusNotFound, err.Error())
	}

	var data map[string]interface{}
	// jsonBytes, err := json.Marshal(dag)
	// if err != nil {
	// 	return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	// }
	// json.Unmarshal(jsonBytes, &data)

	data["dag"] = dag

	return &logical.Response{
		Data: data,
	}, nil
}
