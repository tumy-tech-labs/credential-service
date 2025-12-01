import { Client } from '../../sdk/node/index.js';

async function main() {
  const client = new Client({ baseUrl: 'http://localhost:8080' });

  const issued = await client.issueCredential({
    subject_did: 'did:example:alice',
    ttl_seconds: 600,
    claims: { aud: 'example-api', role: 'admin' }
  });

  const decision = await client.gatewayAuthorize({
    credential: issued.credential,
    expected_audience: 'example-api',
    want_synthetic_jwt: true
  });

  console.log('Subject:', decision.subject);
  console.log('Acting on behalf of:', decision.acting_on_behalf_of || decision.subject);
  console.log('Claims:', decision.claims);
  if (decision.synthetic_jwt) {
    console.log('Synthetic JWT:', decision.synthetic_jwt);
  }
}

main().catch((err) => {
  console.error('example error', err);
  process.exit(1);
});
