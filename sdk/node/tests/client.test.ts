import assert from 'node:assert/strict';
import { test } from 'node:test';
import { CredentialServiceClient, buildUrl } from '../src/client.js';
import { UnauthorizedError } from '../src/errors.js';

const jsonResponse = (body: unknown, status = 200): Response =>
  new Response(JSON.stringify(body), {
    status,
    headers: {
      'Content-Type': 'application/json'
    }
  });

test('buildUrl trims and concatenates paths correctly', () => {
  assert.equal(buildUrl('http://issuer/', '/v1/credentials/issue'), 'http://issuer/v1/credentials/issue');
  assert.equal(buildUrl('http://issuer/', 'v1/credentials/issue'), 'http://issuer/v1/credentials/issue');
});

test('issueVC attaches the format hint and parses responses', async () => {
  const calls: { url: RequestInfo | URL; body?: string | null }[] = [];
  const fetchMock = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    calls.push({ url: input, body: init?.body as string | null });
    return jsonResponse({ credential: 'jwt.token', format: 'jwt-vc' });
  };

  const client = new CredentialServiceClient({
    issuerUrl: 'http://issuer',
    verifierUrl: 'http://verifier',
    fetchImpl: fetchMock
  });

  const resp = await client.issueVC({ subject_did: 'did:example:123' });
  assert.equal(resp.credential, 'jwt.token');

  const parsedBody = JSON.parse(calls[0].body ?? '{}');
  assert.equal(parsedBody.format, 'jwt-vc');
});

test('unauthorized responses raise UnauthorizedError', async () => {
  const fetchMock = async (): Promise<Response> => jsonResponse({ error: 'denied' }, 401);

  const client = new CredentialServiceClient({
    issuerUrl: 'http://issuer',
    verifierUrl: 'http://verifier',
    fetchImpl: fetchMock
  });

  await assert.rejects(client.verify('token'), (err) => {
    assert.ok(err instanceof UnauthorizedError);
    assert.equal(err.message, 'denied');
    return true;
  });
});
