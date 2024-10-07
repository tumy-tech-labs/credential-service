// index.js
const Client = require('./client');
require('dotenv').config();

const main = async () => {
  // Initialize client with DID and Resolver service URLs
  const client = new Client(process.env.DID_SERVICE_URL, process.env.RESOLVER_SERVICE_URL);

  // Organization ID
  const orgID = process.env.ORGANIZATION_ID;

  try {
    // Create a DID
    const didResp = await client.createDID(orgID);
    console.log('DID created successfully:', didResp.id);

    // Resolve the DID
    const resolvedDID = await client.resolveDID(didResp.id);
    console.log('Resolved DID:', resolvedDID);

    // Example of issuing a credential (modify based on your schema)
    const credential = {
      type: 'VerifiableCredential',
      issuer: didResp.id,
      credentialSubject: {
        id: 'did:example:123',
        name: 'John Doe',
      },
      issuanceDate: new Date().toISOString(),
    };
    // not implemented yet
    //const issuedCredential = await client.issueCredential(credential);
    //console.log('Issued Credential:', issuedCredential);
  } catch (error) {
    console.error('Error:', error);
  }
};

main();
