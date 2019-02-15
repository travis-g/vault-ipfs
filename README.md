# vault-ipfs

A Vault plugin for encrypted IPFS interactions.

- Fine-grained controls for authorizing reading and writing encrypted IPFS objects using Vault directly.
- TLS security from client to Vault, then both transit and at-rest encryption within IPFS.
- Audit trails to indicate who attempted to read (and decrypt) Vault-managed IPFS objects.
- An abstraction layer that allows for "versioning" of IPFS objects managed by Vault.

Essentially, this plugin bootstraps Vault's native transit encryption capability and key/value store:

1. Plaintext is sent to Vault to be encrypted.
2. Once it's encrypted, Vault forwards the data to IPFS.
3. Once it's ingested, the IPFS daemon running in the plugin returns the data's hash to Vault.
4. Vault stores the hash in a key/value store for the IPFS mount, and returns the hash to the client.

The client can then discard the hash or record it, but Vault keeps the hash in the catalogue for the mount; at any time, an operator can determine which hashes on IPFS are being managed using a given mount.

When the client wants that data back from IPFS,

1. The client requests the CID through the endpoint,
2. Vault retrieves the requested IPFS object by CID through its local agent,
3. The plugin attempts to decrypt the object's `Data` using the mount's decryption key.

## Versioning

Vault's Key-Value V2 store supports versioning secrets, but IFPS's objects are immutable. By layering Vault over IPFS (carefully), a huge possibility for IPFS opens up. Suppose a client wants to post an "update" to a Vault-managed IPFS CID `/ipfs/Qmabc123`:

1. The client crafts a new IPFS object and `PUT`s the object against `/ipfs/data/Qmabc123`.
2. Vault encrypts the data and uploads the new DAG to IPFS. Lets say the new object is `/ipfs/Qmxyz789`.
3. Vault creates a separate entry to manage the new object at its specific hash: `/ipfs/data/Qmxyz879`.
4. _Internally_, Vault creates a new version of its reference for what it understands as "ref Qmabc123", and pointing the new reference version to "ref Qmxyz789".
5. When a `GET` request for `/ipfs/data/Qmabc123` is made without a requested version, Vault reads, decrypts, and returns `/ipfs/Qmxyz789` by default instead of `/ipfs/Qmabc123`.

Until access is provisioned, reads for `/ipfs/data/Qmxyz789` from Vault directly will be denied, so unless the IPFS object is accessed through Vault at the initial reference of `Qmabc123` the value returned will remain encrypted at rest on the network. A Vault operator can "tidy" these references and their versions later by traversing managed objects, provisioning the appropriate policies and purging the outdated metadata from Vault.

## Limitations

- This plugin can't track add, read, and encryption/decryption attempts made with encryption keys if they're removed from Vault.
- It's impossible to rotate encryption keys in the traditional sense: you can re-encrypt objects with a new key, but objects encrypted with the old key(s) may remain on IPFS forever[\*](#deletion).

### Deletion

Deleting content from The Permanent Web [is complicated](https://github.com/ipfs/faq/issues/9), but theoretically possible. To summarize, if a node not under your influence chooses to replicate an object, you can't force it to take the object down. However, if all nodes that replicated the object decide to unpin or remove the object, the object will eventually fade from IPFS as it's garbage collected.

When the IPFS mesh fails to retrieve data using an object's CID, the CID's content could be considered "deleted", although records in the ledger would prove _something_ existed.

## Policies

As a few examples of what's possible:

```hcl
// Explicitly forbid reading unmanaged IPFS objects.
path "ipfs/object/*" {
  capabilities = ["deny"]
}

// Allow uploading data to IPFS.
path "ipfs/data" {
  capabilities = ["create"]
}

// Allow traversing the managed objects' DAG trees' Links. A policy like this
// can be used for garbage collection of outdated objects' versions.
path "ipfs/metadata" {
  capabilities = ["list"]
}
path "ipfs/metadata/*" {
  capabilities = ["list"]
}

// Allow changing references of managed objects, but only if the DAGs are part
// of the network and Vault-managed (`create` is not allowed) already.
path "ipfs/data/*" {
  capabilities = ["update"]
}

// Allow listing Links of a specific DAG without allowing decryption of the
// object Data.
path "ipfs/data/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG" {
  capabilities = ["list"]
}

// Allow reading a specific DAG link. Reading managed data read using an
// alternate path/the Link's direct hash would be implicitly disallowed.
path "ipfs/data/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/readme" {
  capabilities = ["read"]
}
```

## Caveats

### Object Size

Although Vault won't store the IPFS object data itself, it still needs to process the full Data payload to encrypt or decrypt it. Vault's TLS listener is configured to deny payloads above [32MB]() by default to help mitigate denial-of-service attacks. The maximum size can be adjusted per-listener.

### Encoding

Vault is not meant to process binary data, only key-value pairs. For the sake of consistency data must be base64-encoded prior to being posted against Vault and base64-decoded after being read.

## API

The API was designed to resemble Vault's Key-Value V2 secrets engine API.

### Read IPFS Engine Configuration

This path retreives the current configuration for the IPFS backend at the given path.

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
