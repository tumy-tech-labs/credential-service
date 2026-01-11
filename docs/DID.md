# DID Toolkit

The project ships with a lightweight DID toolkit focused on the `did:jwk` method. A `did:jwk` identifier encodes the public key material directly in the DID string, which makes it easy to bootstrap signing and verification flows without an external DID registry.

## Generating and managing DIDs

The `idctl` CLI can create, export, import, and rotate DIDs using a local JSON store (default: `~/.credential-service/dids.json`). Each entry keeps a small rotation history so you can continue to verify credentials signed with older keys.

```bash
# Create a new DID and persist it (defaults to EdDSA)
idctl did create --label issuer

# Export a DID document to a file
idctl did export --label issuer --output issuer.did.json

# Import an existing DID document
idctl did import --file issuer.did.json --label issuer

# Rotate keys for a stored DID
idctl did rotate --label issuer
```

The stored documents include the DID string, the active key, and prior keys for basic history tracking. If you need to change the storage location, set `DID_STORE_PATH` for the services or pass `--store` to the CLI commands.
