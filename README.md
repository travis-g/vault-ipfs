# vault-ipfs

Essentially, this plugin bootstraps Vault's native transit encryption capability and key/value store:

1. Plaintext is sent to Vault to be encrypted.
2. Once it's encrypted, Vault forwards the data to IPFS.
3. Once it's ingested, the IPFS daemon running in the plugin returns the data's hash to Vault.
4. Vault stores the hash in a key/value store for the IPFS mount, and returns the hash to the client.

The client can then discard the hash or store it, but Vault keeps the hash in the catalogue for the mount; at any time, an operator can determine which hashes on the IPFS were encrypted using a given mount.

When the client wants that data back from IPFS,

1. The client requests the CID through the endpoint,
2. Vault retrieves the requested IPFS object by CID through its local agent,
3. The plugin attempts to decrypt the object's `Data` using the mount's decryption key.

So why use this plugin? What does it give you?

- Fine-grained controls for authorizing reading and writing encrypted IPFS objects using Vault directly.
- TLS security from client to Vault, then both transit and at-rest encryption within IPFS.
- Audit trails: who attempted to read (and decrypt) Vault-encrypted IPFS objects?

## Limitations

- This plugin can't track add, read, and encryption/decryption attempts made with encryption keys if they're removed from Vault.
- It's impossible to rotate encryption keys in the traditional sense: you can re-encrypt objects with a new key, but objects encrypted with the old key(s) may remain on IPFS forever[\*](#deletion).

## Deletion

Deleting _content_ from The Permanent Web [is complicated](https://github.com/ipfs/faq/issues/9), but theoretically possible. To summarize, if a node not under your influence chooses to replicate an object, you can't force it to take the object down. However, if all nodes that replicated the object decide to unpin the object, the _content_ will eventually fade from IPFS as it's garbage collected.

When the IPFS mesh fails to retrieve data using an object's CID, the CID's content could be considered "deleted", although records in the ledger would prove _something_ existed.

## Policies

As an example of what's possible:

```hcl
// Explicitly forbid reading unmanaged IPFS objects
path "ipfs/object/*" {
    capabilities = ["deny"]
}

// Allow uploading data to IPFS
path "ipfs/data" {
    capabilities = ["create"]
}

// Only allow reading links of specific hashes; managed data read using an
// alternate path would be implicitly disallowed, and in that event the
// decryption key would not be applied.
path "ipfs/data/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/*" {
    capabilities = ["list", "read"]
}
```

## Caveats

### File Size

Although Vault won't store the data itself, it still needs to process the full payload to encrypt or decrypt it. Vault's TLS listener is configured to deny payloads above [32MB]() by default to help mitigate denial-of-service attacks. The maximum size can be adjusted per-listener.

### Encoding

Vault is not meant to process binary data, only key-value pairs. Any data must be base64-encoded prior to being posted against Vault and base64-decoded after being read.

### Versioned Key-Values

Due to IPFS's immutable nature there is no concept of versioning.

## API

The API was designed to resemble Vault's Key-Value V2 secrets engine API.

### Get Object

Retrieve an object DAG directly from IPFS.

| Method | Path                 | Produces                 |
| ------ | -------------------- | ------------------------ |
| `GET`  | `/ipfs/object/:hash` | `200 (application/json)` |

#### Parameters

- `hash` `(string: <required>)` - Hash of content to retrieve.

### Add Managed Object

This endpoint encrypts and adds data to IPFS. The calling token must have an ACL policy granting `create` capability.

| Method | Path         | Produces                 |
| ------ | ------------ | ------------------------ |
| `POST` | `/ipfs/data` | `200 (application/json)` |
| `PUT`  | `/ipfs/data` | `200 (application/json)` |

#### Parameters

- `plaintext` `(string: <required>)` - Specifies the base64 encoded object data.
- `encrypt` `(bool: true)` - If true, Vault will encrypt the object data using the backend's public key prior to adding it to IPFS. If false, Vault will add the plaintext to IPFS directly.

### Get Managed Object

This endpoint retrieves a Vault-managed IPFS object's data.

| Method | Path               | Produces                 |
| ------ | ------------------ | ------------------------ |
| `GET`  | `/ipfs/data/:hash` | `200 (application/json)` |

#### Parameters

- `hash` `(string: <required>)` - Hash of content to retrieve and decrypt.
- `decrypt` `(bool: true)` - If true, Vault will decrypt the object data returned by IPFS using the IPFS backend's private key. If false, Vault will return the IPFS object's data as-is.

### Pin Object

Pins an object to the plugin's underlying IPFS daemon.

| Method | Path              | Produces                 |
| ------ | ----------------- | ------------------------ |
| `POST` | `/ipfs/pin/:hash` | `200 (application/json)` |

#### Parameters

- `hash` `(string: <required>)` - Hash of content to pin.

### Unpin Managed Object

Unpins an object from the plugin's underlying IPFS daemon. Unpinned objects are free to be garbage collected at the daemon's discretion.

| Method | Path                | Produces                 |
| ------ | ------------------- | ------------------------ |
| `POST` | `/ipfs/unpin/:hash` | `204 (application/json)` |

#### Parameters

- `hash` `(string: <required>)` - Hash of content to unpin.

### List Managed Objects

This endpoint lists the backend's catalogued IPFS objects. The calling token must have an ACL policy granting `list` capability.

| Method | Path             | Produces                 |
| ------ | ---------------- | ------------------------ |
| `LIST` | `/ipfs/metadata` | `200 (application/json)` |

### Delete Metadata

This endpoint requests that Vault delete metadata for IPFS objects from the catalogue.

Objects can't be explicitly destroyed on "The Permanent Web": it would be misleading to classify this as a `destroy` operation for an object itself, as the data may still persist on IPFS forever.

!~ This does not delete objects from the IPFS network (see [Deletion](#deletion)).

| Method   | Path                   | Produces           |
| -------- | ---------------------- | ------------------ |
| `DELETE` | `/ipfs/metadata/:hash` | `204 (empty body)` |

#### Parameters

- `hash` `(string: <required>)` - Hash of content for which to purge metadata.
