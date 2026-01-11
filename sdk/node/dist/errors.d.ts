export declare class CredentialServiceError extends Error {
    readonly status?: number;
    constructor(message: string, status?: number);
}
export declare class InvalidRequestError extends CredentialServiceError {
}
export declare class UnauthorizedError extends CredentialServiceError {
}
export declare class ServerError extends CredentialServiceError {
}
export declare class NetworkError extends CredentialServiceError {
}
