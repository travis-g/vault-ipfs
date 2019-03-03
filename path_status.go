package ipfs

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	ipfs "github.com/ipfs/go-ipfs-api"
)

func (b *backend) statusPaths() []*framework.Path {
	return []*framework.Path{
		// The order of these paths matters: more specific ones need to be near
		// the top, so that path matching does not short-circuit.
		&framework.Path{
			Pattern:      "status",
			HelpSynopsis: "Return the IPFS backend's status",
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation: b.pathStatusGet,
			},
		},
		&framework.Path{
			Pattern:      "status/peers",
			HelpSynopsis: "Return the IPFS backend node's peer infos",
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ReadOperation: b.pathStatusPeersRead,
				logical.ListOperation: b.pathStatusPeersList,
			},
		},
		&framework.Path{
			Pattern:      "status/peers/",
			HelpSynopsis: "Return the IPFS backend node's peer list",
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ListOperation: b.pathStatusPeersList,
			},
		},
	}
}

type Status struct {
	Peers int `json:"peers"`
}

type StatusPeers struct {
	Peers *ipfs.SwarmConnInfos
}

func (b *backend) pathStatusGet(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	if err := validateFields(req, d); err != nil {
		return nil, logical.CodedError(http.StatusUnprocessableEntity, err.Error())
	}

	sh := ipfs.NewShell(ipfsAddr)

	peers, err := sh.SwarmPeers(ctx)
	if err != nil {
		return nil, logical.CodedError(http.StatusNotFound, err.Error())
	}

	object := &Status{
		Peers: len(peers.Peers),
	}

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

func (b *backend) pathStatusPeersRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	if err := validateFields(req, d); err != nil {
		return nil, logical.CodedError(http.StatusUnprocessableEntity, err.Error())
	}

	sh := ipfs.NewShell(ipfsAddr)

	peers, err := sh.SwarmPeers(ctx)
	if err != nil {
		return nil, logical.CodedError(http.StatusNotFound, err.Error())
	}

	object := StatusPeers{
		Peers: peers,
	}

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

func (b *backend) pathStatusPeersList(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	if err := validateFields(req, d); err != nil {
		return nil, logical.CodedError(http.StatusUnprocessableEntity, err.Error())
	}

	sh := ipfs.NewShell(ipfsAddr)

	peers, err := sh.SwarmPeers(ctx)
	if err != nil {
		return nil, logical.CodedError(http.StatusNotFound, err.Error())
	}

	// Restructure SwarmConnInfos to strings
	peersList := make([]string, 0, len(peers.Peers))
	for _, peer := range peers.Peers {
		infos := []string{
			peer.Addr,
			peer.Peer,
		}
		peersList = append(peersList, strings.Join(infos, "/ipfs/"))
	}

	return logical.ListResponse(peersList), nil
}
