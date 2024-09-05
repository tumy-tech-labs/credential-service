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
- **REST API**: Expose endpoints for creating and retrieving credentials.

## Requirements

- Go 1.19 or higher
- PostgreSQL 12+
- Docker 20+

## Installation

Step-by-step instructions on how to install the project.

1. Clone the repository:

   ```bash
   git clone https://github.com/your-username/credential-service.git
   cd credential-service
   ```

### Request Payload

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

### Example Request

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

### Response

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
  }
}
```

## Configuration

Details about any configuration options (e.g., environment variables, config files).

Environment Variables:

```bash
DATABASE_URL=postgres://cred-service:cred-service-1@postgres:5432/credential-service
PORT=8080
```

## Testing

Instructions for running tests, if applicable.

Example:

bash
Copy code
go test ./...

## Contributing

Explain how others can contribute to the project.

Fork the repository.
Create a new branch (git checkout -b feature-branch).
Commit your changes (git commit -am 'Add new feature').
Push the branch (git push origin feature-branch).
Open a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contact

How to reach you for support or issues.

Email: <your.email@example.com>
GitHub: your-username
