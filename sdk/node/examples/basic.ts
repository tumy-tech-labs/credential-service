import { CredentialServiceClient } from '../src/index.js';

async function main() {
  const client = new CredentialServiceClient({
    issuerUrl: process.env.ISSUER_URL || 'http://localhost:8080',
    verifierUrl: process.env.VERIFIER_URL || 'http://localhost:8081',
    gatewayUrl: process.env.GATEWAY_URL
  });

  const issued = await client.issueVC({
    subject_did: 'did:example:alice',
    ttl_seconds: 600,
    claims: { aud: 'https://api.example.com', scope: ['payments'] }
  });
  console.log('Issued credential', issued);

  const verification = await client.verify(issued.credential);
  console.log('Verification result', verification);

  const decision = await client.authorize({
    credentials: [issued.credential],
    expected_audience: 'https://api.example.com',
    want_synthetic_jwt: true
  });
  console.log('Authorization decision', decision);
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
