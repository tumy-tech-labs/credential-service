class BaseError extends Error {
  constructor(message, status) {
    super(message);
    this.status = status;
  }
}

export class InvalidRequestError extends BaseError {}
export class UnauthorizedError extends BaseError {}
export class ServerError extends BaseError {}
export class NetworkError extends BaseError {}

export class Client {
  constructor({ baseUrl, fetchImpl } = {}) {
    this.baseUrl = baseUrl?.replace(/\/$/, '') || '';
    this.fetchImpl = fetchImpl || globalThis.fetch;
    if (!this.fetchImpl) {
      throw new Error('fetch implementation required');
    }
  }

  async issueCredential(request) {
    return this.#post('/v1/credentials/issue', request);
  }

  async delegateCredential(request) {
    return this.#post('/v1/credentials/delegate', request);
  }

  async verify(token) {
    return this.#post('/v1/credentials/verify', { credential: token });
  }

  async gatewayAuthorize(request) {
    return this.#post('/v1/gateway/authorize', request);
  }

  async #post(path, payload) {
    const url = `${this.baseUrl}${path}`;
    let res;
    try {
      res = await this.fetchImpl(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify(payload || {})
      });
    } catch (err) {
      throw new NetworkError('network_error', err?.status);
    }

    if (!res.ok) {
      throw this.#mapError(res.status);
    }

    const text = await res.text();
    if (!text) return {};
    return JSON.parse(text);
  }

  #mapError(status) {
    if (status === 400) return new InvalidRequestError('invalid_request', status);
    if (status === 401 || status === 403) return new UnauthorizedError('unauthorized', status);
    if (status >= 500) return new ServerError('server_error', status);
    return new NetworkError('network_error', status);
  }
}
