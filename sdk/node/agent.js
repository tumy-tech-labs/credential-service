import { UnauthorizedError } from './client.js';

function decodePayload(token) {
  const parts = token.split('.');
  if (parts.length < 2) throw new Error('invalid token');
  const json = Buffer.from(parts[1], 'base64url').toString('utf8');
  return JSON.parse(json);
}

function scopeFromClaims(claims = {}) {
  const raw = claims.scope;
  if (!raw) return [];
  if (Array.isArray(raw)) return raw.map(String);
  return [String(raw)];
}

function isScopeSubset(parent, child) {
  const set = new Set(parent);
  return child.every((s) => set.has(s));
}

function clampTTL(ttlSeconds, parentExpiresAt) {
  if (!parentExpiresAt) return ttlSeconds;
  const remainingMs = new Date(parentExpiresAt).getTime() - Date.now();
  if (remainingMs <= 0) return 0;
  const remainingSec = Math.floor(remainingMs / 1000);
  if (!ttlSeconds || ttlSeconds > remainingSec) return remainingSec;
  return ttlSeconds;
}

export async function startAgentSession({ client, parentToken, agentDid, scope = [], ttlSeconds }) {
  if (!client) throw new Error('client required');
  if (!parentToken) throw new Error('parent token required');
  const parent = decodePayload(parentToken);
  const parentScope = scopeFromClaims(parent.claims);
  if (!isScopeSubset(parentScope, scope)) throw new UnauthorizedError('scope must be subset');
  const ttl = clampTTL(ttlSeconds, parent.expires_at);
  if (ttl <= 0) throw new UnauthorizedError('ttl exceeded');

  const resp = await client.delegateCredential({
    parent_credential: parentToken,
    delegate_did: agentDid,
    scope,
    ttl_seconds: ttl
  });
  const child = decodePayload(resp.credential || resp.Credential || resp.credential);
  return {
    agentDid,
    accessToken: resp.credential || resp.Credential,
    expiresAt: child.expires_at,
    parentToken,
    scope
  };
}

export async function refreshAgentSession({ client, session }) {
  const ttlSeconds = Math.floor((new Date(session.expiresAt).getTime() - Date.now()) / 1000);
  const fresh = await startAgentSession({ client, parentToken: session.parentToken, agentDid: session.agentDid, scope: session.scope, ttlSeconds });
  Object.assign(session, fresh);
  return session;
}

export async function callAuthorized({ client, session, method, url, body }) {
  if (new Date(session.expiresAt).getTime() <= Date.now()) {
    await refreshAgentSession({ client, session });
  }
  const decision = await client.gatewayAuthorize({
    credentials: [session.parentToken, session.accessToken],
    want_synthetic_jwt: true
  });
  if (!decision.allowed) throw new UnauthorizedError(decision.reason || 'denied');

  const res = await client.fetchImpl(url, {
    method,
    headers: { Authorization: `Bearer ${decision.synthetic_jwt}` },
    body: body ? JSON.stringify(body) : undefined
  });
  return res;
}
