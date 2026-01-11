export class CredentialServiceError extends Error {
  readonly status?: number;

  constructor(message: string, status?: number) {
    super(message);
    this.name = this.constructor.name;
    this.status = status;
  }
}

export class InvalidRequestError extends CredentialServiceError {}
export class UnauthorizedError extends CredentialServiceError {}
export class ServerError extends CredentialServiceError {}
export class NetworkError extends CredentialServiceError {}
