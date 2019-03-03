package ipfs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"

	"github.com/fatih/structs"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
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
var dagPutFields = map[string]*framework.FieldSchema{
	"plaintext": &framework.FieldSchema{
		Type:        framework.TypeString,
		Description: "base64 encoded plaintext data to upload",
	},
	"format": &framework.FieldSchema{
		Type:        framework.TypeString,
		Default:     "dag-cbor",
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

func (b *backend) dagPaths() []*framework.Path {
	return []*framework.Path{
		// The order of these paths matters: more specific ones need to be near
		// the top, so that path matching does not short-circuit.
		&framework.Path{
			Pattern:      "dag/" + framework.GenericNameRegex("ref") + framework.OptionalParamRegex("link") + "/",
			HelpSynopsis: "Return an IPLD node's links",
			Fields:       dagFields,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ListOperation: b.pathDAGList,
			},
		},
		&framework.Path{
			Pattern:      "dag/" + framework.GenericNameRegex("ref") + framework.OptionalParamRegex("link"),
			HelpSynopsis: "Return an IPLD node",
			Fields:       dagFields,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation: b.pathDAGGet,
			},
		},
		&framework.Path{
			Pattern:      "dag/" + framework.GenericNameRegex("ref") + "/",
			HelpSynopsis: "Return a list of a node's links",
			Fields:       dagFields,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ListOperation: b.pathDAGList,
			},
		},
		&framework.Path{
			Pattern:      "dag",
			HelpSynopsis: "Return a list of a node's links",
			Fields:       dagPutFields,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.CreateOperation: b.pathDAGPut,
				logical.UpdateOperation: b.pathDAGPut,
			},
		},
	}
}

type DAGLinks struct {
	Links []Link `json:"links"`
}

type Link struct {
	Name string
	Cid  Cid
}

type Cid struct {
	Target string `json:"/"`
}

// pathDAGGet returns an IPLD node as returned by the network.
func (b *backend) pathDAGGet(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	key := d.Get("ref").(string)
	link := d.Get("link").(string)
	if link != "" {
		key = strings.Join([]string{key, link}, "/")
	}

	addr := fmt.Sprintf("%s/api/v0/dag/get?arg=%s", ipfsAddr, key)

	httpReq, err := http.NewRequest(http.MethodGet, addr, nil)
	httpReq.Header.Set("User-Agent", "vault-plugin-ipfs")
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	res, err := ipfsClient.Do(httpReq)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)

	var data map[string]interface{}
	json.Unmarshal(body, &data)

	return &logical.Response{
		Data: data,
	}, nil
}

func (b *backend) pathDAGList(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	key := d.Get("ref").(string)
	link := d.Get("link").(string)
	if link != "" {
		key = strings.Join([]string{key, link}, "/")
	}

	if key == "" {
		return logical.ErrorResponse("Missing ref"), nil
	}

	addr := fmt.Sprintf("%s/api/v0/dag/get?arg=%s", ipfsAddr, key)

	httpReq, err := http.NewRequest(http.MethodGet, addr, nil)
	httpReq.Header.Set("User-Agent", "vault-plugin-ipfs")
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	res, err := ipfsClient.Do(httpReq)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)

	var data DAGLinks
	json.Unmarshal(body, &data)

	// Pull just the links out of the node
	hashes := make([]string, 0, len(data.Links))
	for _, link := range data.Links {
		hashes = append(hashes, link.Name)
	}

	sort.Strings(hashes)

	return logical.ListResponse(hashes), nil
}

func (b *backend) pathDAGPut(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	plaintext := d.Get("plaintext").(string)
	format := d.Get("format").(string)
	encoding := d.Get("input-enc").(string)
	pin := d.Get("pin").(bool)
	hash := d.Get("hash").(string)

	if plaintext == "" {
		return logical.ErrorResponse("No plaintext provided"), nil
	}

	addr := fmt.Sprintf("%s/api/v0/dag/put?format=%s&input-enc=%s&pin=%t&hash=%s", ipfsAddr, format, encoding, pin, hash)

	requestBody := &bytes.Buffer{}
	writer := multipart.NewWriter(requestBody)
	err := writer.WriteField("file", plaintext)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}
	// close multipart writer before sending request
	err = writer.Close()
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	httpReq, err := http.NewRequest(http.MethodPost, addr, requestBody)
	httpReq.Header.Set("User-Agent", "vault-plugin-ipfs")
	httpReq.Header.Set("Content-Type", "multipart/form-data")

	res, err := ipfsClient.Do(httpReq)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}
	defer res.Body.Close()

	var byteResponse []byte
	res.Body.Read(byteResponse)

	var obj map[string]interface{}
	json.Unmarshal(byteResponse, &obj)
	if res.StatusCode != http.StatusOK {
		return logical.ErrorResponse(string(byteResponse)), nil
	}

	var cid Cid

	return &logical.Response{
		Data: structs.Map(cid),
	}, nil
}
