// SD-JWT example using local sdk-nodejs helpers
const Client = require('../../../sdk-nodejs/client');

async function run() {
  const issuerURL = process.env.ISSUER_URL || 'http://localhost:8080';
  const verifierURL = process.env.VERIFIER_URL || 'http://localhost:8081';
  const subject_did = process.env.ALICE_DID;
  if (!subject_did) {
    console.error('ALICE_DID is required. Generate via ./bin/keygen -did-only and export ALICE_DID before running.');
    process.exit(1);
  }

  const c = new Client({ issuerURL, verifierURL });
  const issued = await c.issueSdJwtCredential({
    subject_did,
    ttl_seconds: 600,
    claims: { email: 'alice@example.com', department: 'engineering', scope: 'read:orders' }
  });
  console.log('SD-JWT:', issued.credential);
  console.log('Disclosures:', issued.disclosures);

  const verify = await c.verifySdJwtCredential({
    credential: issued.credential,
    disclosures: [issued.disclosures[0]]
  });
  console.log('Partial verification active:', verify.active);
}

run().catch(err => console.error(err.response?.data || err.message));
