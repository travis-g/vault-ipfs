# vault-ipfs

A Vault plugin for data and access management on IPFS.

- Fine-grained controls for authorizing reading and writing asymmetrically encrypted IPFS objects.
- TLS security from client to Vault, then encryption of data in-transit and at rest within IPFS.
- Audit trails to indicate who attempted to access IPFS Merkle forest data using Vault as a proxy.

<!-- - An abstraction layer that allows for "versioning" of IPFS objects managed by Vault. -->

Essentially, this plugin was inspired by Vault's native asymmetric transit encryption capability and core key-value store:

1. Plaintext is sent to Vault to be encrypted,
2. Once it's encrypted, Vault creates and forwards DAGs to IPFS,
3. Once it's ingested, the IPFS API returns the data's Content Identifier hash to Vault,
4. Vault stores the hash in its key-value store for the IPFS mount, and returns the hash to the client for reference.

The client can then discard the hash or record it, but Vault keeps the hash in a catalogue for the mount; at any time, an operator can determine which hashes on IPFS are being managed using a given mount.

When a client wants Vault-managed encrypted data from IPFS,

1. The client requests the encrypted DAG's hash through through Vault,
2. Vault retrieves the requested IPFS object by hash through IPFS's API,
3. The plugin decrypts the DAG's data using the mount's decryption key.

## Clever Merkle Forest Joke

The public IPFS Merkle forest is immense, and provisioning policies for individual DAGs of Merkle trees would lead to complex and unmaintainable policies. Additionally, if a Merkle DAG's hash is known, it can be queried through a public IPFS node and its full tree can be discovered.

<!-- Is there reverse-lookup to discover what links to a DAG? -->

Consider the well-known IPFS docs DAG `/ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG`:

```console
% ipfs ls QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG
QmZTR5bcpQD7cFgTorqxZDYaew1Wqgfbd2ud9QqGPAkK2V 1688 about
QmYCvbfNbCwFR45HiNP45rwJgvatpiW38D961L5qAhUM5Y 200  contact
QmY5heUM5qgRubMDD1og9fhCPA6QdkMp3QCwd4s7gJsyE7 322  help
QmdncfsVm2h5Kqq9hPmU7oAVX2zTSVP3L869tgTbPYnsha 1728 quick-start
QmPZ9gcCEpqKTo6aq61g2nXGUhM4iCL3ewB6LDXZCtioEB 1102 readme
QmTumTjvcYCAvRRwQ8sDRxh8ezmrcr88YFU7iYNroGGTBZ 1027 security-notes
```

To allow clients to read the IPFS tree through Vault, `read` and `list` capability could be provisioned to the initial DAG, and access provisioned to all linked objects explicitly. We could create a Vault policy with distinct `path "ipfs/object/<hash>"` stanzas for each CID in the tree, totaling 7 stanzas to provision access.

A much more maintainable solution supported by the plugin is to provision `read` and `list` on `path "ipfs/object/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/*"`: authorized clients can list the initial tree's links and read the data beneath them as far down as the Merkle tree extends. Using the `readme` link specifically, the globbed path allows implicit `read` and `list` access to `/ipfs/object/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/readme` without creating explicit policy grants for the `readme`'s alternate `/ipfs/object/QmPZ9gcCEpqKTo6aq61g2nXGUhM4iCL3ewB6LDXZCtioEB` path.

<!-- Restriction policies -->

## Versioning Immutable Objects

Vault's key-value V2 store supports versioning secrets, but objects in IFPS's Merkle forest are immutable. By layering Vault as a gateway over IPFS (carefully), a huge possibility for IPFS DAG meta-versioning opens up. Suppose a client is authorized to post an "update" to a Vault-managed IPFS DAG `/ipfs/Qmabc123`:

1. The client crafts a new IPFS DAG and `PUT`s the data against Vault at `/ipfs/data/Qmabc123`.
2. Vault encrypts the data and uploads the new DAG to IPFS. Lets say the new DAG is `/ipfs/Qmxyz789`.
3. _In its catalogue_, Vault creates a new version of `ref Qmabc123`, and points this new version to `ref Qmxyz789`.
4. When a `GET` request for `/ipfs/data/Qmabc123` is made without a requested version, Vault reads its catalogue, discovers `ref Qmxyz789`, and gets, decrypts, and returns IPFS DAG `/ipfs/Qmxyz789` by default instead of `/ipfs/Qmabc123`.

Until access is provisioned, reads for `/ipfs/data/Qmxyz789` from Vault directly will be denied, so unless the IPFS object is accessed through Vault at the initial reference of `Qmabc123` the value returned will remain encrypted at rest on the network. A Vault operator can "tidy" these references and their versions later by traversing managed objects, importing new DAGs directly to the catalogue, provisioning the appropriate policy updates and purging the outdated metadata from Vault.

## Warnings

- Inter-IFPS node communication uses custom encryption that has not yet been audited. The mechanism is not SSL or TLS, but there is community discussion around [implementing TLS 1.3]().
- This plugin can't record encryption/decryption attempts made with using a backend's encryption keys if they are removed from Vault.
- It's impossible to rotate encryption keys for Vault-managed IPFS data in the traditional sense: you can re-encrypt objects with a new key, but objects encrypted with the old key(s) may remain on IPFS forever[\*](#deletion).
- Content stored on IPFS is not natively encrypted at rest. However, this plugin mitigates that problem by asymmetrically encrypting the data it will manage prior to uploading.
- File and directory names are uploaded to IPFS in plaintext.

## Policies

Building on the Merkle tree isolation and versioning explanations above, here are policy examples of what's possible:

```hcl
// Explicitly forbid using Vault as a proxy to read unmanaged IPFS objects.
path "ipfs/object/*" {
  capabilities = ["deny"]
}

// Allow uploading data to IPFS.
path "ipfs/data" {
  capabilities = ["create"]
}

// Allow traversing the managed objects' DAG trees' metadata. A policy like this
// can be used for client-initiated garbage collection of outdated objects'
// version references.
path "ipfs/metadata" {
  capabilities = ["list"]
}
path "ipfs/metadata/*" {
  capabilities = ["list"]
}

// Allow changing references of managed objects, but only if the DAGs are part
// of the network and Vault-managed already (`create` is not allowed).
path "ipfs/data/*" {
  capabilities = ["update"]
}

// Allow listing links of a specific DAG without allowing decryption of the
// object data.
path "ipfs/data/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG" {
  capabilities = ["list"]
}

// Same as above, but allow enumeration of a specific DAG's entire Merkle tree.
// The policy allows listing of linked DAG's links, and their links, etc.
path "ipfs/data/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/*" {
  capabilities = ["list"]
}

// Allow reading a specific link of a DAG. Reading managed data using an
// alternate path, such as from `/ipfs/data/<readme-hash>` would be implicitly
// disallowed.
path "ipfs/data/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/readme" {
  capabilities = ["read"]
}

// Allow access to a full managed DAG's Merkle tree, but forbid access to a DAG
// linked within it. The referenced link cannot be accessed (read and decrypted)
// through Vault unless permissions are circumvented with access to another tree
// that contains the node or by a policy on the DAG's direct hash.
path "ipfs/data/Qmb8wsGZNXt5VXZh1pEmYynjB6Euqpq3HYyeAdw2vScTkQ/*" {
  capabilities = ["read", "list"]
}
path "ipfs/data/Qmb8wsGZNXt5VXZh1pEmYynjB6Euqpq3HYyeAdw2vScTkQ/838 - Incident/*" {
  capabilities = ["deny"]
}
```

## Caveats

### Deletion

Deleting content from The Permanent Web [is complicated](https://github.com/ipfs/faq/issues/9), but theoretically possible. To summarize, if a node not under your influence pins or replicates an object, you can't force it to take the object down. However, if all nodes that replicated the object decide to unpin it, garbage collect it, or go offline, the object will eventually fade from IPFS.

When the IPFS mesh fails to retrieve data using an object's CID, the CID's content could be considered "deleted", although the ledger would prove _something_ existed.

### Object Size

Although Vault won't store the IPFS object data itself, it still needs to process the full Data payload to encrypt or decrypt it. Vault's TCP listeners are configured to deny payloads above 32MB by default to help mitigate denial-of-service attacks. The maximum size can be adjusted per-listener.

The plugin's API does not support pulling a full Merkle tree in a single request, but if individual DAGs requested through Vault surpass 32MB in size the [`max_request_size`](https://www.vaultproject.io/docs/configuration/listener/tcp.html#max_request_size) parameter can be adjusted.

### Encoding

Vault is not meant to process binary data, only key-value pairs. For the sake of consistency data must be base64 encoded prior to being posted against Vault and base64 decoded after being read.

## API

The API was designed to resemble Vault's Key-Value V2 secrets engine API, and it does not account for all of IPFS's capabilities. IPFS's API itself is very full-featured, but is not yet stable.

### Read IPFS Engine Configuration

This path retrieves the current configuration for the IPFS backend at the given path.

| Method | Path           | Produces                 |
| ------ | -------------- | ------------------------ |
| `GET`  | `/ipfs/config` | `200 (application/json)` |

### Get Object

Retrieve an object DAG directly from IPFS.

| Method | Path                         | Produces                 |
| ------ | ---------------------------- | ------------------------ |
| `GET`  | `/ipfs/object/:hash(/:link)` | `200 (application/json)` |

#### Parameters

- `hash` `(string: <required>)` - Hash of content to retrieve.
- `link` `(string: <optional>)` - Link of the desired hash's DAG to return.

### List Object Links

Retrieve the list of `Links` of an object DAG directly from IPFS.

| Method | Path                         | Produces                 |
| ------ | ---------------------------- | ------------------------ |
| `LIST` | `/ipfs/object/:hash(/:link)` | `200 (application/json)` |

#### Parameters

- `hash` `(string: <required>)` - Hash of content to retrieve DAG links.
- `link` `(string: <optional>)` - Link of the desired hash's DAG to query.

<!-- TODO:

### Add Managed Object

This endpoint encrypts and adds data to IPFS. The calling token must have an ACL policy granting `create` capability.

| Method | Path         | Produces                 |
| ------ | ------------ | ------------------------ |
| `POST` | `/ipfs/data` | `200 (application/json)` |
| `PUT`  | `/ipfs/data` | `200 (application/json)` |

#### Parameters

- `plaintext` `(string: <required>)` - Specifies the base64 encoded object data.

### Update Managed Object

This endpoint uploads the provided data to IPFS and aliases it as an existing hash under Vault's management. If the new object does not exist in IPFS already, the calling token must have an ACL policy granting `create` and `update` capabilities. If the value already exists in IPFS the calling token needs only `update` capabilities. The new object is added as a new version of the targeted object.

| Method | Path               | Produces                 |
| ------ | ------------------ | ------------------------ |
| `PUT`  | `/ipfs/data/:hash` | `200 (application/json)` |

#### Parameters

- `hash` `(string: <required>)` - Hash of object to update in Vault's store.
- `plaintext` `(string: <required>)` - Specifies the base64 encoded object data.

### Get Managed Object

This endpoint retrieves a Vault-managed IPFS object's data.

| Method | Path               | Produces                 |
| ------ | ------------------ | ------------------------ |
| `GET`  | `/ipfs/data/:hash` | `200 (application/json)` |

#### Parameters

- `decrypt` `(bool: true)` - If true, Vault will decrypt the object data returned by IPFS using the IPFS backend's private key. If false, Vault will return the IPFS object's data as-is.
- `hash` `(string: <required>)` - Hash of content to retrieve and decrypt.
- `version` `(int: 0)` - Specifies the version to return. The latest version will be returned if not set.

### Delete Latest Version of Object

This endpoint soft deletes the object's latest version. This marks the version as deleted, but the underlying object data will not be removed from Vault.

| Method   | Path               | Produces                 |
| -------- | ------------------ | ------------------------ |
| `DELETE` | `/ipfs/data/:hash` | `200 (application/json)` |

#### Parameters

- `hash` `(string: <required>)` - Hash of content to delete.

### Delete Object Versions

This endpoint soft deletes specific versions of an object.

| Method | Path                 | Produces                 |
| ------ | -------------------- | ------------------------ |
| `POST` | `/ipfs/delete/:hash` | `200 (application/json)` |

#### Parameters

- `hash` `(string: <required>)` - Hash of content to retrieve and decrypt.
- `versions` `([]int: <required>)` - The versions to be deleted. The data will not be deleted, but it will no longer be returned.

### List Managed Objects

This endpoint lists the backend's catalogued IPFS objects. The calling token must have an ACL policy granting `list` capability.

| Method | Path             | Produces                 |
| ------ | ---------------- | ------------------------ |
| `LIST` | `/ipfs/metadata` | `200 (application/json)` |

### Read Metadata

| Method | Path                   | Produces                 |
| ------ | ---------------------- | ------------------------ |
| `GET`  | `/ipfs/metadata/:hash` | `200 (application/json)` |

#### Parameters

- `hash` `(string: <required>)` - Hash of content for which to retrieve metadata.

### Delete Metadata and All Versions

This endpoint requests that Vault delete metadata for IPFS objects from the catalogue.

Objects can't be explicitly destroyed on "The Permanent Web": it would be misleading to classify this as a `destroy` operation for an object itself, as the data may still persist on IPFS forever.

!~ This does not delete objects from the IPFS network (see [Deletion](#deletion)).

| Method   | Path                   | Produces           |
| -------- | ---------------------- | ------------------ |
| `DELETE` | `/ipfs/metadata/:hash` | `204 (empty body)` |

#### Parameters

- `hash` `(string: <required>)` - Hash of content for which to purge metadata.

### Pin Object

Pins an object to the plugin's underlying IPFS daemon's local storage.

| Method | Path              | Produces                 |
| ------ | ----------------- | ------------------------ |
| `POST` | `/ipfs/pin/:hash` | `200 (application/json)` |

#### Parameters

- `hash` `(string: <required>)` - Hash of content to pin.

### Unpin Managed Object

Unpins an object from the plugin's underlying IPFS daemon's local storage. Unpinned objects are free to be garbage collected at the daemon's discretion.

| Method | Path                | Produces                 |
| ------ | ------------------- | ------------------------ |
| `POST` | `/ipfs/unpin/:hash` | `204 (application/json)` |

#### Parameters

- `hash` `(string: <required>)` - Hash of content to unpin.

-->

## Links

- [Archives on IPFS](https://archives.ipfs.io/)

## License

:memo:
