"""Main client for the Credential Service SDK."""

import os
from dataclasses import asdict
from datetime import datetime
from typing import Optional, Union
from urllib.parse import urljoin

try:
    import httpx
    HTTPX_AVAILABLE = True
except ImportError:
    HTTPX_AVAILABLE = False

try:
    import requests
    REQUESTS_AVAILABLE = True
except ImportError:
    REQUESTS_AVAILABLE = False

from .exceptions import (
    InvalidRequestError,
    NetworkError,
    ServerError,
    UnauthorizedError,
)
from .types import (
    AuthorizeRequest,
    AuthorizeResponse,
    DelegateRequest,
    DelegateResponse,
    GatewayAgentContext,
    IssueRequest,
    IssueResponse,
    VerifyRequest,
    VerifyResponse,
)


class CredentialServiceClient:
    """Client for interacting with Credential Service APIs.

    Supports both httpx (async/sync) and requests (sync) HTTP libraries.
    Prefers httpx if available, falls back to requests.

    Args:
        issuer_url: Base URL for the issuer service (default: http://localhost:8080)
        verifier_url: Base URL for the verifier service (default: http://localhost:8081)
        gateway_url: Base URL for the gateway service (defaults to verifier_url)
        timeout: Request timeout in seconds (default: 15)
        http_client: Optional pre-configured HTTP client (httpx.Client or requests.Session)

    Examples:
        >>> # Using environment variables
        >>> client = CredentialServiceClient.from_env()

        >>> # Using local development defaults
        >>> client = CredentialServiceClient.local_dev()

        >>> # Custom configuration
        >>> client = CredentialServiceClient(
        ...     issuer_url="https://issuer.example.com",
        ...     verifier_url="https://verifier.example.com"
        ... )
    """

    def __init__(
        self,
        issuer_url: str = "http://localhost:8080",
        verifier_url: str = "http://localhost:8081",
        gateway_url: Optional[str] = None,
        timeout: float = 15.0,
        http_client: Optional[Union["httpx.Client", "requests.Session"]] = None,
    ):
        """Initialize the Credential Service client."""
        self.issuer_url = issuer_url.rstrip("/")
        self.verifier_url = verifier_url.rstrip("/")
        self.gateway_url = gateway_url.rstrip("/") if gateway_url else self.verifier_url
        self.timeout = timeout

        # Determine HTTP client
        if http_client:
            self._http_client = http_client
            self._owns_client = False
        elif HTTPX_AVAILABLE:
            self._http_client = httpx.Client(timeout=timeout)
            self._owns_client = True
        elif REQUESTS_AVAILABLE:
            self._http_client = requests.Session()
            self._owns_client = True
        else:
            raise ImportError(
                "Neither httpx nor requests is installed. "
                "Install one: pip install httpx or pip install requests"
            )

    @classmethod
    def from_env(cls) -> "CredentialServiceClient":
        """Create a client using environment variables.

        Reads from:
        - ISSUER_URL
        - VERIFIER_URL
        - GATEWAY_URL
        """
        return cls(
            issuer_url=os.getenv("ISSUER_URL", "http://localhost:8080"),
            verifier_url=os.getenv("VERIFIER_URL", "http://localhost:8081"),
            gateway_url=os.getenv("GATEWAY_URL"),
        )

    @classmethod
    def local_dev(cls) -> "CredentialServiceClient":
        """Create a client configured for local development (docker-compose)."""
        return cls(
            issuer_url="http://localhost:8080",
            verifier_url="http://localhost:8081",
            gateway_url="http://localhost:8081",
        )

    def __enter__(self):
        """Context manager entry."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit - close HTTP client if we own it."""
        if self._owns_client:
            self.close()

    def close(self):
        """Close the underlying HTTP client."""
        if self._owns_client and hasattr(self._http_client, "close"):
            self._http_client.close()

    def issue_vc(self, request: IssueRequest) -> IssueResponse:
        """Issue a standard VC-JWT credential.

        Args:
            request: Issue request with subject_did, ttl_seconds, and claims

        Returns:
            IssueResponse containing the signed credential

        Raises:
            InvalidRequestError: Request validation failed
            UnauthorizedError: Authentication/authorization failed
            ServerError: Server-side error
            NetworkError: Network or transport error
        """
        request.format = "jwt-vc"
        return self._issue(request)

    def issue_sd_jwt(self, request: IssueRequest) -> IssueResponse:
        """Issue an SD-JWT credential with selective disclosure.

        Args:
            request: Issue request with subject_did, ttl_seconds, and claims

        Returns:
            IssueResponse containing credential and disclosures array
        """
        request.format = "sd-jwt"
        return self._issue(request)

    def _issue(self, request: IssueRequest) -> IssueResponse:
        """Internal method to issue credentials."""
        data = self._post(self.issuer_url, "/v1/credentials/issue", request)
        return IssueResponse(**data)

    def verify(
        self,
        credential: Optional[str] = None,
        request: Optional[VerifyRequest] = None
    ) -> VerifyResponse:
        """Verify a credential or credential chain.

        Args:
            credential: Single credential string (convenience parameter)
            request: Full VerifyRequest for advanced use cases

        Returns:
            VerifyResponse with verification details

        Examples:
            >>> # Simple verification
            >>> result = client.verify(credential="eyJ...")

            >>> # Verify chain with expected audience
            >>> result = client.verify(
            ...     request=VerifyRequest(
            ...         credentials=["eyJ...", "eyJ..."],
            ...         expected_audience="sample-api"
            ...     )
            ... )
        """
        if request is None:
            if credential is None:
                raise InvalidRequestError("Either credential or request must be provided")
            request = VerifyRequest(credential=credential)

        data = self._post(self.verifier_url, "/v1/credentials/verify", request)

        # Parse expires_at if present
        if "expires_at" in data:
            if isinstance(data["expires_at"], str):
                data["expires_at"] = datetime.fromisoformat(data["expires_at"].replace("Z", "+00:00"))

        # Parse agent context if present
        if "agent" in data and data["agent"]:
            data["agent"] = GatewayAgentContext(**data["agent"])

        return VerifyResponse(**data)

    def authorize(self, request: AuthorizeRequest) -> AuthorizeResponse:
        """Request an authorization decision from the gateway.

        Args:
            request: Authorization request with credentials, audience, etc.

        Returns:
            AuthorizeResponse with allow/deny decision and optional synthetic JWT

        Examples:
            >>> # Basic authorization
            >>> decision = client.authorize(
            ...     AuthorizeRequest(
            ...         credential="eyJ...",
            ...         expected_audience="sample-api",
            ...         resource="orders",
            ...         action="read"
            ...     )
            ... )
            >>> if decision.allowed:
            ...     print(f"Access granted: {decision.synthetic_jwt}")
        """
        data = self._post(self.gateway_url, "/v1/gateway/authorize", request)

        # Parse agent context if present
        if "agent" in data and data["agent"]:
            data["agent"] = GatewayAgentContext(**data["agent"])

        return AuthorizeResponse(**data)

    def delegate_credential(self, request: DelegateRequest) -> DelegateResponse:
        """Delegate a credential with constrained scope and TTL.

        Args:
            request: Delegation request with parent credential and delegate DID

        Returns:
            DelegateResponse containing the delegated credential

        Examples:
            >>> # Delegate with reduced scope
            >>> delegated = client.delegate_credential(
            ...     DelegateRequest(
            ...         parent_credential="eyJ...",
            ...         delegate_did="did:jwk:...",
            ...         scope=["read:orders"],
            ...         ttl_seconds=300
            ...     )
            ... )
        """
        data = self._post(self.issuer_url, "/v1/credentials/delegate", request)
        return DelegateResponse(**data)

    def _post(self, base_url: str, path: str, payload: Union[dict, object]) -> dict:
        """Internal POST helper.

        Args:
            base_url: Service base URL
            path: API path
            payload: Request payload (dict or dataclass)

        Returns:
            Response data as dict

        Raises:
            Various CredentialServiceError subclasses
        """
        if not base_url:
            raise InvalidRequestError("Base URL is required")

        url = urljoin(base_url.rstrip("/") + "/", path.lstrip("/"))

        # Convert dataclass to dict if needed
        if hasattr(payload, "__dataclass_fields__"):
            json_data = asdict(payload)
            # Remove None values for cleaner requests
            json_data = {k: v for k, v in json_data.items() if v is not None}
        else:
            json_data = payload

        headers = {
            "Accept": "application/json",
            "Content-Type": "application/json",
        }

        try:
            if HTTPX_AVAILABLE and isinstance(self._http_client, httpx.Client):
                response = self._http_client.post(url, json=json_data, headers=headers)
            else:  # requests
                response = self._http_client.post(
                    url,
                    json=json_data,
                    headers=headers,
                    timeout=self.timeout
                )
        except Exception as e:
            raise NetworkError(f"Network error: {e}")

        # Handle error responses
        if response.status_code >= 400:
            self._handle_error_response(response)

        # Parse response
        try:
            return response.json()
        except Exception:
            # Handle empty or non-JSON responses
            return {}

    def _handle_error_response(self, response):
        """Parse error response and raise appropriate exception."""
        status_code = response.status_code

        # Try to extract error message from response
        error_msg = ""
        try:
            error_data = response.json()
            error_msg = error_data.get("error") or error_data.get("description") or ""
        except Exception:
            error_msg = response.text or f"HTTP {status_code}"

        # Map status code to exception type
        if status_code == 400:
            raise InvalidRequestError(error_msg or "Invalid request", status_code)
        elif status_code in (401, 403):
            raise UnauthorizedError(error_msg or "Unauthorized", status_code)
        elif status_code >= 500:
            raise ServerError(error_msg or "Server error", status_code)
        else:
            raise NetworkError(error_msg or f"HTTP {status_code}", status_code)
