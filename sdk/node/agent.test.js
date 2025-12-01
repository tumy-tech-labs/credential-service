import { Client } from './index.js';
import { startAgentSession, callAuthorized } from './agent.js';

function buildToken(payload) {
  const header = Buffer.from(JSON.stringify({ alg: 'none' })).toString('base64url');
  const body = Buffer.from(JSON.stringify(payload)).toString('base64url');
  return `${header}.${body}.sig`;
}

async function run() {
  const now = Date.now();
  const parent = buildToken({ subject: 'did:parent', expires_at: new Date(now + 600000).toISOString(), claims: { scope: ['read'] } });
  let delegateCalls = 0;
  const client = new Client({
    baseUrl: 'http://localhost',
    fetchImpl: async (url, opts) => {
      if (url.endsWith('/v1/credentials/delegate')) {
        delegateCalls++;
        const payload = JSON.parse(opts.body);
        return {
          ok: true,
          status: 200,
          text: async () => JSON.stringify({ credential: buildToken({ subject: 'did:agent', expires_at: new Date(now + 300000).toISOString(), claims: { scope: payload.scope } }) })
        };
      }
      if (url.endsWith('/v1/gateway/authorize')) {
        return { ok: true, status: 200, text: async () => JSON.stringify({ allowed: true, synthetic_jwt: 'syn' }) };
      }
      return { ok: true, status: 200, text: async () => '{}' };
    }
  });

  const session = await startAgentSession({ client, parentToken: parent, agentDid: 'did:agent', scope: ['read'], ttlSeconds: 120 });
  session.expiresAt = new Date(now - 1000).toISOString();
  await callAuthorized({ client, session, method: 'GET', url: 'http://service/resource' });
  console.log('delegate calls', delegateCalls);
}

run().catch((err) => {
  console.error(err);
  process.exit(1);
});
