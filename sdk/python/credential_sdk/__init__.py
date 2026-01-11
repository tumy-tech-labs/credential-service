"""Credential Service Python SDK.

A modern Python client for issuing, verifying, and authorizing W3C Verifiable Credentials
with the Credential Service.
"""

from .client import CredentialServiceClient
from .types import (
    IssueRequest,
    IssueResponse,
    VerifyRequest,
    VerifyResponse,
    AuthorizeRequest,
    AuthorizeResponse,
    DelegateRequest,
    DelegateResponse,
    GatewayAgentContext,
)
from .exceptions import (
    CredentialServiceError,
    NetworkError,
    InvalidRequestError,
    UnauthorizedError,
    ServerError,
)

__version__ = "0.1.0"

__all__ = [
    "CredentialServiceClient",
    "IssueRequest",
    "IssueResponse",
    "VerifyRequest",
    "VerifyResponse",
    "AuthorizeRequest",
    "AuthorizeResponse",
    "DelegateRequest",
    "DelegateResponse",
    "GatewayAgentContext",
    "CredentialServiceError",
    "NetworkError",
    "InvalidRequestError",
    "UnauthorizedError",
    "ServerError",
]
