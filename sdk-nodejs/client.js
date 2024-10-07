// client.js
const axios = require('axios');
require('dotenv').config();

class Client {
  constructor(didServiceURL, resolverServiceURL) {
    this.didServiceURL = didServiceURL || process.env.DID_SERVICE_URL;
    this.resolverServiceURL = resolverServiceURL || process.env.RESOLVER_SERVICE_URL;
  }

  // Create a DID
  async createDID(orgID) {
    try {
      const payload = {
        type: 'organization',
        organization_id: orgID,
      };
      const response = await axios.post(`${this.didServiceURL}/dids`, payload);
      return response.data;
    } catch (error) {
      console.error('Error creating DID:', error.response ? error.response.data : error.message);
      throw error;
    }
  }

  // Resolve a DID
  async resolveDID(did) {
    try {
      const response = await axios.get(`${this.resolverServiceURL}/dids/resolver`, {
        params: { did },
      });
      return response.data;
    } catch (error) {
      console.error('Error resolving DID:', error.response ? error.response.data : error.message);
      throw error;
    }
  }

  // Issue a credential
  async issueCredential(credential) {
    try {
      const response = await axios.post(`${this.didServiceURL}/v1/credential`, credential);
      return response.data;
    } catch (error) {
      console.error('Error issuing credential:', error.response ? error.response.data : error.message);
      throw error;
    }
  }
}

module.exports = Client;
