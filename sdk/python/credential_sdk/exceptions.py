"""Exceptions for the Credential Service SDK."""


class CredentialServiceError(Exception):
    """Base exception for all Credential Service SDK errors."""

    def __init__(self, message: str, status_code: int = None):
        super().__init__(message)
        self.message = message
        self.status_code = status_code


class NetworkError(CredentialServiceError):
    """Network or transport-level error."""

    pass


class InvalidRequestError(CredentialServiceError):
    """Request was invalid or malformed."""

    pass


class UnauthorizedError(CredentialServiceError):
    """Request was unauthorized or forbidden."""

    pass


class ServerError(CredentialServiceError):
    """Server-side error occurred."""

    pass
