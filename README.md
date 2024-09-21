# credential-service

A service for creating and managing Verifiable Credentials, including associating them with Decentralized Identifiers (DIDs).

## Table of Contents

1. [Features](#features)
2. [Requirements](#requirements)
3. [Installation](#installation)
4. [Usage](#usage)
5. [Configuration](#configuration)
6. [Testing](#testing)
7. [Contributing](#contributing)
8. [License](#license)
9. [Contact](#contact)

## Features

- **Create Verifiable Credentials**: Issue credentials with unique IDs, expiration dates, and digital signatures.
- **Manage DIDs**: Integrate with the DID management service to use existing DIDs as issuers.
- **Dynamic Payloads**: Include issuer DID and subject details in the POST request payload.
- **REST API**: Expose endpoints for creating, retrieving, and revoking credentials.

```mermaid
sequenceDiagram
    participant Issuer
    participant DIDService
    participant IssuerService
    participant Holder
    participant PresentationService
    participant Verifier
    participant VerificationService

    Issuer->>DIDService: POST /dids (Create DID for Holder)
    DIDService-->>Issuer: Return DID
    Issuer->>IssuerService: POST /credentials (Issue Credential with Holder DID)
    IssuerService-->>Issuer: Return Credential
    Issuer->>Holder: Send Credential
    Holder->>PresentationService: POST /presentations (Create Presentation with Credential)
    PresentationService-->>Holder: Return Presentation ID
    Holder->>Verifier: Send Presentation ID
    Verifier->>VerificationService: GET /presentations/{id} (Request Presentation)
    VerificationService->>PresentationService: Fetch Presentation Data
    PresentationService-->>VerificationService: Return Presentation Data
    VerificationService->>DIDService: GET /dids/resolver (Resolve DID)
    DIDService-->>VerificationService: Return DID Document
    VerificationService->>VerificationService: Verify Presentation
    VerificationService-->>Verifier: Return Verification Result
    Verifier->>Holder: Return Verification Status
```

## Requirements

- Go 1.19 or higher
- PostgreSQL 12+
- Docker 20+

## Installation

Step-by-step instructions on how to install the project.

1. Clone the repository:

   ```bash
   git clone https://github.com/bradtumy/credential-service.git
   cd credential-service
   ```

## Usage

1. Start up in Docker
  
  ```bash
  docker compose up -d --build
  ```

### Create DID

**Request:**
```bash
curl -X POST http://localhost:8080/dids \
-H "Content-Type: application/json" \
-d '{
  "holder": "did:key:z6MholderDIDhere"
}'
```

**Response:**
```json
{
  "did": "did:key:z6MnewDIDhere"
}
```

### Resolve DID

**Request:**
```bash
curl -X GET http://localhost:8080/dids/resolver?did=did:key:z6MnewDIDhere
```

**Response:**
```json
{
  "did": "did:key:z6MnewDIDhere",
  "document": { ... } // DID Document details
}
```

### Issue Credentials

**Request Payload:**
```json
{
  "issuerDid": "did:key:z6MyourIssuerDIDhere",
  "subject": {
    "name": "Jane Doe",
    "email": "jane.doe@example.com",
    "phone": "+3214567890"
  }
}
```

**Example Request:**
```bash
curl -X POST http://localhost:8080/credentials \
-H "Content-Type: application/json" \
-d '{
  "issuerDid": "did:key:z6MyourIssuerDIDhere",
  "subject": {
    "name": "Jane Doe",
    "email": "jane.doe@example.com",
    "phone": "+3214567890"
  }
}'
```

**Response:**
```json
{
  "@context": "https://www.w3.org/2018/credentials/v1",
  "id": "credential-id",
  "type": ["VerifiableCredential", "EmploymentCredential"],
  "issuer": "did:key:z6MyourIssuerDIDhere",
  "issuanceDate": "2024-09-05T00:00:00Z",
  "expirationDate": "2025-09-05T00:00:00Z",
  "credentialSubject": {
    "id": "did:key:z6MsubjectDIDhere",
    "name": "Jane Doe",
    "email": "jane.doe@example.com",
    "phone": "+3214567890"
  },
  "signature": "i9CASyhzQD1nDL/gpzacq0etT5jCAuH7MPHYU9WA7p/0yrirtD2Y4Mdg8G8dEr6kMgqenpWt5MP/5MaYkhUtDg=="
}
```

### Get All Credentials

**Request:**
```bash
curl -X GET http://localhost:8080/credentials
```

**Response:**
```json
[
  {
    "id": "credential-id",
    "issuer": "did:key:z6MyourIssuerDIDhere",
    "subject": { ... },
    "issuanceDate": "2024-09-05T00:00:00Z",
    "expirationDate": "2025-09-05T00:00:00Z",
    "signature": "signature-value"
  },
  ...
]
```

### Revoke Credentials

**Request:**
```bash
curl -X DELETE http://localhost:8080/credentials/credential-id
```

**Response:**
```json
{
  "message": "Credential revoked successfully."
}
```

### Create Presentation

**Request:**
```bash
curl -X POST http://localhost:8080/presentations \
-H "Content-Type: application/json" \
-d '{
  "holderDid": "did:key:z6MholderDIDhere",
  "credentials": ["credential-id"]
}'
```

**Response:**
```json
{
  "presentationId": "presentation-id"
}
```

### Get Presentation

**Request:**
```bash
curl -X GET http://localhost:8080/presentations/presentation-id
```

**Response:**
```json
{
  "presentationId": "presentation-id",
  "credentials": [ ... ]
}
```

### Verification Service

**Request:**
```bash
curl -X GET http://localhost:8080/verifications/presentation-id
```

**Response:**
```json
{
  "presentationId": "presentation-id",
  "verificationStatus": "verified"
}
```

### Testing the Vault Connection

To test the connection to Vault, you can run the following command inside your Docker container:

```bash
curl -X GET http://localhost:8200/v1/sys/health \
-H "X-Vault-Token: your-vault-token"
```

Replace `your-vault-token` with your actual Vault token. A successful response will confirm that the Vault service is healthy.

## Configuration

Details about any configuration options (e.g., environment variables, config files).

**Environment Variables:**

```bash
DATABASE_URL=postgres://cred-service:cred-service-1@postgres:5432/credential-service
PORT=8080
VAULT_ADDR=http://vault:8200
VAULT_TOKEN=your-vault-token
```

## Testing

Instructions for running tests, if applicable.

**Example:**
```bash
go test ./...
```

## Contributing

How others can contribute to the project.

1. Fork the repository.
2. Create a new branch (git checkout -b feature-branch).
3. Commit your changes (git commit -am 'Add new feature').
4. Push the branch (git push origin feature-branch).
5. Open a Pull Request.

## License

This project is licensed under the Apache2 License - see the LICENSE file for details.

## Contact

How to connect for support or issues.

Email: <brad@tumy-tech.com>  
GitHub: bradtumy
