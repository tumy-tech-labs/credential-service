import {
  AuthorizeRequest,
  AuthorizeResponse,
  CredentialServiceOptions,
  DelegateRequest,
  DelegateResponse,
  FetchLike,
  IssueRequest,
  IssueResponse,
  VerifyRequest,
  VerifyResponse
} from './types.js';
import { InvalidRequestError, NetworkError, ServerError, UnauthorizedError } from './errors.js';

export function buildUrl(baseUrl: string | undefined, path: string): string {
  if (!baseUrl) {
    throw new InvalidRequestError('base URL is required');
  }
  const normalizedBase = baseUrl.replace(/\/$/, '');
  if (path.startsWith('/')) {
    return `${normalizedBase}${path}`;
  }
  return `${normalizedBase}/${path}`;
}

export class CredentialServiceClient {
  readonly issuerUrl?: string;
  readonly verifierUrl?: string;
  readonly gatewayUrl?: string;
  readonly fetchImpl: FetchLike;

  constructor({ issuerUrl, verifierUrl, gatewayUrl, fetchImpl }: CredentialServiceOptions = {}) {
    const chosenFetch = fetchImpl ?? globalThis.fetch;
    if (!chosenFetch) {
      throw new Error('fetch implementation required');
    }

    this.issuerUrl = issuerUrl?.replace(/\/$/, '');
    this.verifierUrl = verifierUrl?.replace(/\/$/, '');
    this.gatewayUrl = gatewayUrl?.replace(/\/$/, '');
    this.fetchImpl = chosenFetch;
  }

  async issueVC(request: IssueRequest): Promise<IssueResponse> {
    return this.issue({ ...request, format: 'jwt-vc' });
  }

  async issueSDJWT(request: IssueRequest): Promise<IssueResponse> {
    return this.issue({ ...request, format: 'sd-jwt' });
  }

  async verify(credentialOrRequest: string | VerifyRequest): Promise<VerifyResponse> {
    const payload: VerifyRequest =
      typeof credentialOrRequest === 'string'
        ? { credential: credentialOrRequest }
        : credentialOrRequest;

    return this.post(this.verifierUrl, '/v1/credentials/verify', payload);
  }

  async authorize(request: AuthorizeRequest): Promise<AuthorizeResponse> {
    const base = this.gatewayUrl || this.verifierUrl;
    return this.post(base, '/v1/gateway/authorize', request);
  }

  async delegateCredential(request: DelegateRequest): Promise<DelegateResponse> {
    return this.post(this.issuerUrl, '/v1/credentials/delegate', request);
  }

  private async issue(request: IssueRequest): Promise<IssueResponse> {
    return this.post(this.issuerUrl, '/v1/credentials/issue', request);
  }

  private async post<TInput, TOutput>(baseUrl: string | undefined, path: string, payload: TInput): Promise<TOutput> {
    const url = buildUrl(baseUrl, path);

    let response: Response;
    try {
      response = await this.fetchImpl(url, {
        method: 'POST',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(payload ?? {})
      });
    } catch (err: any) {
      throw new NetworkError('network_error', err?.status);
    }

    if (!response.ok) {
      throw await this.mapError(response);
    }

    const text = await response.text();
    if (!text) return {} as TOutput;
    try {
      return JSON.parse(text) as TOutput;
    } catch (err) {
      throw new ServerError('invalid_json_response');
    }
  }

  private async mapError(response: Response): Promise<Error> {
    const status = response.status;
    let errorText = '';
    try {
      const body = await response.json();
      errorText = body?.error || body?.description || '';
    } catch (_) {
      // Ignore parse errors; fall back to status-based errors.
    }

    if (status === 400) return new InvalidRequestError(errorText || 'invalid_request', status);
    if (status === 401 || status === 403) return new UnauthorizedError(errorText || 'unauthorized', status);
    if (status >= 500) return new ServerError(errorText || 'server_error', status);
    return new NetworkError(errorText || 'network_error', status);
  }
}

export default CredentialServiceClient;
