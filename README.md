# Credential Service

A robust microservice designed for creating, managing, and verifying **W3C-compliant Verifiable Credentials (VCs)**. This service allows organizations to issue credentials, link them to **Decentralized Identifiers (DIDs)**, and enable secure, privacy-preserving verification across multiple platforms.

## Key Features

- **Verifiable Credentials**: Issue credentials for a wide range of use cases, including identity verification, employment, education, and more, with full compliance to **W3C standards**.
- **Decentralized Identifiers (DIDs)**: Associate VCs with **DIDs**, enabling portable and self-sovereign identities that users control.
- **Interoperability & Portability**: Supports cross-industry use, allowing credentials to be reused and verified across different platforms without re-registration or data duplication.
- **Privacy by Design**: Minimal disclosure and secure storage, ensuring that only necessary information is shared during verifications.
- **Revocation Support**: Includes mechanisms to revoke credentials, ensuring that only valid credentials can be verified.
- **Verifiable Presentations**: Generate and verify presentations of credentials to prove claims in a trusted and decentralized manner.
- **Microservice Architecture**: Built as part of a modular and containerized microservice ecosystem, easily integratable into broader decentralized identity solutions.
- **Issuer, Holder, Verifier Roles**: The service supports a complete lifecycle of identity management by addressing the needs of issuers, credential holders, and verifiers.

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

Sequence Diagram:

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

Data Flow Diagram:

```mermaid
graph TD
    IssuerService["Issuer Service"]
    HolderService["Holder Service"]
    VerifierService["Verifier Service"]
    ResolverService["Resolver Service"]
    DIDService["DID Service"]

    subgraph DID Creation
        IssuerService -->|Creates DID| DIDService
    end

    subgraph Credential Issuance
        IssuerService -->|Issues Credential| HolderService
    end

    subgraph Credential Verification
        HolderService -->|Presents Credential| VerifierService
    end

    VerifierService -->|Queries| ResolverService
    IssuerService -->|Queries| ResolverService
    DIDService -->|Stores DID Document| ResolverService

    ResolverService -->|Resolves DID| IssuerService
    ResolverService -->|Resolves DID| VerifierService
```

## Requirements

- Go 1.19 or higher
- PostgreSQL 12+
- Docker 20+

## Installation

Step-by-step instructions on how to install the project.

1. Clone the repository:

  ```bash
  git clone https://github.com/tumy-tech-labs/credential-service.git
  cd credential-service
  ```

## Usage

1. Start up in Docker
  
  ```bash
  docker compose up -d --build
  ```

### Create DID

This is currently limited to creating a DID for Issuers.  When creating the DID it will generate a private and public key and store the private key in a hashicorp vault.

TODO: Create DID's for Holders and Verifiers as well.

**Request:**

```bash
curl -X POST http://localhost:8080/dids \
-H "Content-Type: application/json" \
-d '{
  "organization_id": "org123"
}'
```

**Response:**

```json
{
    "@context": "https://www.w3.org/ns/did/v1",
    "id": "did:key:z6M52fX64ItBn_w-GybPu9P6U3-kOO1F5MSpCfHrheKb0k",
    "publicKey": [
        {
            "id": "did:key:z6M52fX64ItBn_w-GybPu9P6U3-kOO1F5MSpCfHrheKb0k#keys-1",
            "type": "Ed25519VerificationKey2018",
            "controller": "did:key:z6M52fX64ItBn_w-GybPu9P6U3-kOO1F5MSpCfHrheKb0k",
            "publicKeyBase58": "52fX64ItBn_w-GybPu9P6U3-kOO1F5MSpCfHrheKb0k"
        }
    ],
    "createdAt": "2024-10-07T22:47:10Z",
    "organization_id": "orgABC"
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

When Issuing the credentials, provide the DID that you created in the previous steps.

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

This is currently not working.

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

## Holder Service

The Holder Service is a microservice responsible for receiving, storing, and presenting verifiable credentials issued by the Issuer Service. It ensures compliance with W3C standards and provides an API for interaction with the credentials.

- **Receive Credentials**: Accepts verifiable credentials from the Issuer Service and stores them in memory.
- **Present Credentials**: Allows users to present stored credentials for verification to third parties.
- **Validation**: Validates incoming credentials to ensure they meet required standards.

### API Endpoints

#### 1. Receive Credential

- **Endpoint**: `/holder/receive`
- **Method**: `POST`
- **Description**: Receives a verifiable credential and stores it in memory.
- **Request Body**:
  
  ```json
  {
    "@context": [
      "https://www.w3.org/2018/credentials/v1"
    ],
    "type": [
      "VerifiableCredential"
    ],
    "id": "string",
    "issuer": "string",
    "issuanceDate": "string",
    "expirationDate": "string",
    "credentialSubject": {
      "email": "string",
      "id": "string",
      "name": "string",
      "phone": "string"
    },
    "proof": {
      "type": "string",
      "created": "string",
      "proofValue": "string",
      "proofPurpose": "string",
      "verificationMethod": "string"
    }
  }
  ```

#### 2. Present Credential

- **Endpoint**: `/holder/present`
- **Method**: `GET`
- **Description**: Presents all stored credentials for verification.
- **Response**:
  - Returns an array of stored credentials.

### Getting Started

1. **Run the Holder Service**:
Ensure Docker is running and use the following command to start the service:

   ```bash
   docker-compose up --build
   ```

2. **Test the API**:
Use tools like `curl` or Postman to interact with the API:

- To receive a credential:

     ```bash
     curl -X POST http://localhost:8082/holder/receive -d '{"your":"data"}' -H "Content-Type: application/json"
     ```

- To present stored credentials:
  
     ```bash
     curl -X GET http://localhost:8082/holder/present
     ```

### Notes

- The Holder Service currently stores credentials in memory for simplicity. Future implementations may include persistent storage.
- Ensure that the service adheres to W3C standards for verifiable credentials.

## Testing

Instructions for running tests, if applicable.

**Example:**

```bash
go test ./...
```

Here's an updated section for the `README.md` that covers the new Verifier service:

---

## Verifier Service

The **Verifier Service** is responsible for validating Verifiable Presentations (VPs) received from the Holder Service. It ensures the authenticity and integrity of the Verifiable Credentials (VCs) within the VP and verifies the proofs attached by both the issuer and the holder. This service adheres to the W3C standards for Verifiable Credentials and Verifiable Presentations.

### Key Features

- Accepts Verifiable Presentations (VPs) from the Holder.
- Validates the signature (proof) from both the holder and issuer.
- Checks the integrity of the Verifiable Credential (VC), including expiration and issuer authenticity.
- Built as a microservice to integrate into the credential verification ecosystem.

### API Endpoints

#### 1. `POST /verifier/verify`

This endpoint receives a Verifiable Presentation (VP) from the Holder service and validates the included Verifiable Credentials.

- **Request**
  The payload is a JSON object containing a Verifiable Presentation (VP). The VP includes one or more Verifiable Credentials (VCs) issued by an issuer and presented by the holder.
  
  Example Request Payload:

  ```json
  {
    "@context": ["https://www.w3.org/2018/credentials/v1"],
    "type": ["VerifiablePresentation"],
    "verifiableCredential": [
      {
        "@context": ["https://www.w3.org/2018/credentials/v1"],
        "type": ["VerifiableCredential"],
        "id": "977a9b78-f48b-4d50-a847-10b33b802877",
        "issuer": "did:key:z6MsjAXl18xWy-kgyxn2OJiu3EhSCd6-mnWWztrxhD_1w4",
        "issuanceDate": "2024-09-21T21:57:25Z",
        "expirationDate": "2025-09-21T21:57:25Z",
        "credentialSubject": {
          "id": "did:example:1dab09d7-9dee-4013-8812-fc401794c672",
          "email": "johndoe@example.com",
          "name": "John Doe",
          "phone": "+1234567890"
        },
        "proof": {
          "type": "Ed25519Signature2018",
          "created": "2024-09-21T21:57:25Z",
          "proofValue": "U0lHTkFUVVJF",
          "proofPurpose": "assertionMethod",
          "verificationMethod": "did:key:z6MsjAXl18xWy-kgyxn2OJiu3EhSCd6-mnWWztrxhD_1w4#keys-1"
        }
      }
    ],
    "holder": "did:key:z6MsjAXl18xWy-kgyxn2OJiu3EhSCd6-mnWWztrxhD_1w4",
    "proof": {
      "type": "Ed25519Signature2018",
      "created": "2024-09-21T22:05:30Z",
      "proofPurpose": "authentication",
      "verificationMethod": "did:key:z6MsjAXl18xWy-kgyxn2OJiu3EhSCd6-mnWWztrxhD_1w4#keys-1",
      "proofValue": "SOME_PROOF_SIGNATURE"
    }
  }
  ```

- **Response**:
  The response will indicate whether the presentation and credentials were successfully verified or not.

  Example Success Response:

  ```json
  {
    "status": "success",
    "message": "Verifiable Presentation and Credentials are valid."
  }
  ```

  Example Failure Response:

  ```json
  {
    "status": "error",
    "message": "Invalid signature or credential."
  }
  ```

### Running the Verifier Service

1. **Starting the Service**:
   You can start the verifier service using Docker by building the container as specified in the `docker-compose.yml` file. The service listens on port `8083` by default.

2. **Sending a Verification Request**:
To send a Verifiable Presentation for verification, you can use `curl` or any HTTP client:

   ```bash
   curl -X POST http://localhost:8083/verifier/verify \
   -H "Content-Type: application/json" \
   -d '{
       "@context": ["https://www.w3.org/2018/credentials/v1"],
       "type": ["VerifiablePresentation"],
       "verifiableCredential": [ ... ],
       "holder": "did:key:holderDID",
       "proof": { ... }
   }'
   ```

3. **Debugging**:

The verifier service includes basic logging for tracking verification attempts and their outcomes. Use the logs to troubleshoot failed verifications.

### W3C Compliance

The Verifier Service is designed in compliance with the [W3C Verifiable Credentials](https://www.w3.org/TR/vc-data-model/) and [Verifiable Presentations](https://www.w3.org/TR/vc-data-model/#presentations-0) standards.

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
GitHub: tumy-tech-labs
