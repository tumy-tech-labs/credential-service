# idctl CLI

`idctl` is a lightweight command-line helper for interacting with the credential platform during development. It wraps the issuer and verifier endpoints and a simple DID generator so you can mint and validate credentials without wiring up a UI.

## Building

```bash
make idctl
```

The binary is written to `./bin/idctl`.

## Usage

Generate a new DID and key material:

```bash
./bin/idctl did create
```

Issue a VC-JWT for a subject DID (defaults to `CRED_ISSUER_URL` or `http://localhost:8080`):

```bash
./bin/idctl vc issue --subject-did="did:jwk:alice" --scope="read:orders" --ttl-seconds=600
```

Verify a credential string (defaults to `CRED_VERIFIER_URL` or `http://localhost:8081`):

```bash
./bin/idctl vc verify --credential="<credential>"
```

You can also load the credential from a file with `--credential-file=./token.txt`.
