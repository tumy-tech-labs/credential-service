const crypto = require('crypto');

/**
 * DIDKeyPair represents a generated DID with its associated key material.
 * @typedef {Object} DIDKeyPair
 * @property {string} did - The did:jwk identifier
 * @property {string} algorithm - EdDSA or ES256
 * @property {string} publicJWK - Public key in JWK format (JSON string)
 * @property {string} privateKeyPEM - Private key in PEM format
 */

/**
 * Generate a new DID:JWK with embedded public key.
 * 
 * @param {string} algorithm - Signing algorithm: 'EdDSA' (Ed25519) or 'ES256' (ECDSA P-256)
 * @returns {Promise<DIDKeyPair>} The generated DID and key material
 * 
 * @example
 * const { generateDIDJWK } = require('./keygen');
 * 
 * // Generate EdDSA key
 * const keypair = await generateDIDJWK('EdDSA');
 * console.log('DID:', keypair.did);
 * 
 * // Save private key to file if needed
 * const fs = require('fs');
 * fs.writeFileSync('key.pem', keypair.privateKeyPEM, { mode: 0o600 });
 */
async function generateDIDJWK(algorithm = 'EdDSA') {
  if (algorithm === 'EdDSA') {
    return generateEd25519DID();
  } else if (algorithm === 'ES256') {
    return generateES256DID();
  } else {
    throw new Error(`Unsupported algorithm: ${algorithm} (supported: EdDSA, ES256)`);
  }
}

/**
 * Generate an Ed25519 key pair and format as did:jwk.
 * @private
 */
function generateEd25519DID() {
  return new Promise((resolve, reject) => {
    try {
      // Generate Ed25519 key pair
      const { publicKey, privateKey } = crypto.generateKeyPairSync('ed25519', {
        publicKeyEncoding: {
          type: 'spki',
          format: 'der'
        },
        privateKeyEncoding: {
          type: 'pkcs8',
          format: 'pem'
        }
      });

      // Extract raw public key (last 32 bytes of SPKI format)
      const rawPublicKey = publicKey.slice(-32);

      // Create public JWK
      const publicJWK = {
        kty: 'OKP',
        crv: 'Ed25519',
        x: base64urlEncode(rawPublicKey)
      };

      const publicJWKString = JSON.stringify(publicJWK);

      // Create DID by base64url-encoding the public JWK
      const did = 'did:jwk:' + base64urlEncode(Buffer.from(publicJWKString));

      resolve({
        did,
        algorithm: 'EdDSA',
        publicJWK: publicJWKString,
        privateKeyPEM: privateKey
      });
    } catch (error) {
      reject(new Error(`Failed to generate Ed25519 key: ${error.message}`));
    }
  });
}

/**
 * Generate an ECDSA P-256 key pair and format as did:jwk.
 * @private
 */
function generateES256DID() {
  return new Promise((resolve, reject) => {
    try {
      // Generate ECDSA P-256 key pair
      const { publicKey, privateKey } = crypto.generateKeyPairSync('ec', {
        namedCurve: 'prime256v1', // P-256
        publicKeyEncoding: {
          type: 'spki',
          format: 'der'
        },
        privateKeyEncoding: {
          type: 'pkcs8',
          format: 'pem'
        }
      });

      // Parse the DER-encoded public key to extract x and y coordinates
      // ECDSA P-256 public key in uncompressed format: 0x04 || x || y
      // The uncompressed point is in the last 65 bytes of the SPKI structure
      const publicKeyObject = crypto.createPublicKey({
        key: publicKey,
        format: 'der',
        type: 'spki'
      });

      // Export as JWK to get x and y
      const jwk = publicKeyObject.export({ format: 'jwk' });

      // Create public JWK
      const publicJWK = {
        kty: 'EC',
        crv: 'P-256',
        x: jwk.x,
        y: jwk.y
      };

      const publicJWKString = JSON.stringify(publicJWK);

      // Create DID by base64url-encoding the public JWK
      const did = 'did:jwk:' + base64urlEncode(Buffer.from(publicJWKString));

      resolve({
        did,
        algorithm: 'ES256',
        publicJWK: publicJWKString,
        privateKeyPEM: privateKey
      });
    } catch (error) {
      reject(new Error(`Failed to generate ES256 key: ${error.message}`));
    }
  });
}

/**
 * Base64url encode a buffer without padding.
 * @private
 */
function base64urlEncode(buffer) {
  return buffer.toString('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=/g, '');
}

module.exports = {
  generateDIDJWK
};
