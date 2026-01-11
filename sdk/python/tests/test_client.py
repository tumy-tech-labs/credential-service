"""Unit tests for the CredentialServiceClient."""

import json
from datetime import datetime
from unittest.mock import Mock, patch

import pytest

from credential_sdk import (
    AuthorizeRequest,
    AuthorizeResponse,
    CredentialServiceClient,
    DelegateRequest,
    InvalidRequestError,
    IssueRequest,
    IssueResponse,
    NetworkError,
    ServerError,
    UnauthorizedError,
    VerifyRequest,
    VerifyResponse,
)


class MockResponse:
    """Mock HTTP response."""

    def __init__(self, json_data, status_code=200, text=None):
        self.json_data = json_data
        self.status_code = status_code
        self.text_data = text or json.dumps(json_data)

    def json(self):
        """Return JSON data."""
        if isinstance(self.json_data, Exception):
            raise self.json_data
        return self.json_data

    @property
    def text(self):
        """Return text data."""
        return self.text_data


@pytest.fixture
def mock_http_client():
    """Create a mock HTTP client."""
    return Mock()


@pytest.fixture
def client(mock_http_client):
    """Create a client with mocked HTTP."""
    return CredentialServiceClient(
        issuer_url="http://issuer.test",
        verifier_url="http://verifier.test",
        gateway_url="http://gateway.test",
        http_client=mock_http_client,
    )


def test_client_initialization():
    """Test client initialization with different methods."""
    # Explicit URLs
    client = CredentialServiceClient(
        issuer_url="http://issuer.example.com",
        verifier_url="http://verifier.example.com",
    )
    assert client.issuer_url == "http://issuer.example.com"
    assert client.verifier_url == "http://verifier.example.com"
    assert client.gateway_url == "http://verifier.example.com"

    # Local dev defaults
    client = CredentialServiceClient.local_dev()
    assert client.issuer_url == "http://localhost:8080"
    assert client.verifier_url == "http://localhost:8081"


def test_issue_vc(client, mock_http_client):
    """Test issuing a VC-JWT credential."""
    mock_response = MockResponse({
        "credential": "eyJhbGci...",
        "format": "jwt-vc",
        "api_version": "v1"
    })
    mock_http_client.post.return_value = mock_response

    request = IssueRequest(
        subject_did="did:jwk:test",
        ttl_seconds=600,
        claims={"test": True}
    )
    result = client.issue_vc(request)

    assert isinstance(result, IssueResponse)
    assert result.credential == "eyJhbGci..."
    assert result.format == "jwt-vc"
    mock_http_client.post.assert_called_once()


def test_issue_sd_jwt(client, mock_http_client):
    """Test issuing an SD-JWT credential."""
    mock_response = MockResponse({
        "credential": "eyJhbGci...",
        "disclosures": ["WyJzYWx0...", "WyJzYWx0..."],
        "format": "sd-jwt",
        "api_version": "v1"
    })
    mock_http_client.post.return_value = mock_response

    request = IssueRequest(
        subject_did="did:jwk:test",
        ttl_seconds=600,
        claims={"email": "test@example.com"}
    )
    result = client.issue_sd_jwt(request)

    assert isinstance(result, IssueResponse)
    assert result.credential == "eyJhbGci..."
    assert len(result.disclosures) == 2
    assert result.format == "sd-jwt"


def test_verify_credential(client, mock_http_client):
    """Test verifying a credential."""
    mock_response = MockResponse({
        "valid": True,
        "subject": "did:jwk:test",
        "issuer": "did:jwk:issuer",
        "expires_at": "2025-12-06T21:30:00Z",
        "delegation_depth": 0,
        "claims": {"test": True}
    })
    mock_http_client.post.return_value = mock_response

    result = client.verify(credential="eyJhbGci...")

    assert isinstance(result, VerifyResponse)
    assert result.valid is True
    assert result.subject == "did:jwk:test"
    assert isinstance(result.expires_at, datetime)


def test_verify_with_request(client, mock_http_client):
    """Test verifying with a VerifyRequest."""
    mock_response = MockResponse({
        "valid": True,
        "subject": "did:jwk:test",
        "expires_at": "2025-12-06T21:30:00Z",
        "delegation_depth": 1,
        "acting_on_behalf_of": "did:jwk:parent"
    })
    mock_http_client.post.return_value = mock_response

    request = VerifyRequest(
        credentials=["parent_cred", "child_cred"],
        expected_audience="test-api"
    )
    result = client.verify(request=request)

    assert result.valid is True
    assert result.delegation_depth == 1
    assert result.acting_on_behalf_of == "did:jwk:parent"


def test_authorize(client, mock_http_client):
    """Test gateway authorization."""
    mock_response = MockResponse({
        "allowed": True,
        "subject": "did:jwk:test",
        "delegation_depth": 0,
        "reason": "policy_allow",
        "synthetic_jwt": "eyJhbGci...",
        "policy_id": "policy-1"
    })
    mock_http_client.post.return_value = mock_response

    request = AuthorizeRequest(
        credential="eyJhbGci...",
        expected_audience="test-api",
        resource="orders",
        action="read",
        want_synthetic_jwt=True
    )
    result = client.authorize(request)

    assert isinstance(result, AuthorizeResponse)
    assert result.allowed is True
    assert result.synthetic_jwt == "eyJhbGci..."
    assert result.policy_id == "policy-1"


def test_delegate_credential(client, mock_http_client):
    """Test credential delegation."""
    mock_response = MockResponse({
        "credential": "eyJhbGci...",
        "api_version": "v1"
    })
    mock_http_client.post.return_value = mock_response

    request = DelegateRequest(
        parent_credential="eyJhbGci...",
        delegate_did="did:jwk:agent",
        scope=["read:orders"],
        ttl_seconds=300
    )
    result = client.delegate_credential(request)

    assert result.credential == "eyJhbGci..."


def test_error_handling_400(client, mock_http_client):
    """Test handling of 400 Bad Request."""
    mock_response = MockResponse(
        {"error": "invalid_subject_did"},
        status_code=400
    )
    mock_http_client.post.return_value = mock_response

    with pytest.raises(InvalidRequestError) as exc_info:
        client.issue_vc(IssueRequest(subject_did="invalid", ttl_seconds=600))

    assert exc_info.value.status_code == 400
    assert "invalid_subject_did" in str(exc_info.value.message)


def test_error_handling_401(client, mock_http_client):
    """Test handling of 401 Unauthorized."""
    mock_response = MockResponse(
        {"error": "unauthorized"},
        status_code=401
    )
    mock_http_client.post.return_value = mock_response

    with pytest.raises(UnauthorizedError) as exc_info:
        client.verify(credential="eyJhbGci...")

    assert exc_info.value.status_code == 401


def test_error_handling_500(client, mock_http_client):
    """Test handling of 500 Server Error."""
    mock_response = MockResponse(
        {"error": "internal_server_error"},
        status_code=500
    )
    mock_http_client.post.return_value = mock_response

    with pytest.raises(ServerError) as exc_info:
        client.issue_vc(IssueRequest(subject_did="did:jwk:test", ttl_seconds=600))

    assert exc_info.value.status_code == 500


def test_network_error(client, mock_http_client):
    """Test handling of network errors."""
    mock_http_client.post.side_effect = Exception("Connection refused")

    with pytest.raises(NetworkError):
        client.issue_vc(IssueRequest(subject_did="did:jwk:test", ttl_seconds=600))


def test_context_manager():
    """Test using client as context manager."""
    with CredentialServiceClient.local_dev() as client:
        assert client is not None
        assert client.issuer_url == "http://localhost:8080"


def test_invalid_base_url(client, mock_http_client):
    """Test error when base URL is empty."""
    client.issuer_url = ""

    with pytest.raises(InvalidRequestError) as exc_info:
        client.issue_vc(IssueRequest(subject_did="did:jwk:test", ttl_seconds=600))

    assert "Base URL is required" in str(exc_info.value.message)


def test_verify_without_credential_or_request(client):
    """Test that verify requires either credential or request."""
    with pytest.raises(InvalidRequestError) as exc_info:
        client.verify()

    assert "credential or request must be provided" in str(exc_info.value.message)
