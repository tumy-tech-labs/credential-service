"""Tests for type definitions."""

from datetime import datetime

import pytest

from credential_sdk import (
    AuthorizeRequest,
    AuthorizeResponse,
    DelegateRequest,
    GatewayAgentContext,
    IssueRequest,
    IssueResponse,
    VerifyRequest,
    VerifyResponse,
)


def test_issue_request():
    """Test IssueRequest dataclass."""
    req = IssueRequest(
        subject_did="did:jwk:test",
        ttl_seconds=600,
        claims={"test": True},
        format="jwt-vc"
    )
    assert req.subject_did == "did:jwk:test"
    assert req.ttl_seconds == 600
    assert req.claims == {"test": True}
    assert req.format == "jwt-vc"


def test_issue_response():
    """Test IssueResponse dataclass."""
    resp = IssueResponse(
        credential="eyJhbGci...",
        disclosures=["WyJzYWx0..."],
        format="sd-jwt",
        api_version="v1"
    )
    assert resp.credential == "eyJhbGci..."
    assert len(resp.disclosures) == 1
    assert resp.format == "sd-jwt"


def test_verify_request():
    """Test VerifyRequest dataclass."""
    req = VerifyRequest(
        credential="eyJhbGci...",
        expected_audience="test-api"
    )
    assert req.credential == "eyJhbGci..."
    assert req.expected_audience == "test-api"

    # Test with credentials list
    req2 = VerifyRequest(
        credentials=["cred1", "cred2"],
        format="sd-jwt",
        disclosures=["disc1"]
    )
    assert len(req2.credentials) == 2
    assert req2.format == "sd-jwt"


def test_verify_response():
    """Test VerifyResponse dataclass."""
    now = datetime.now()
    resp = VerifyResponse(
        valid=True,
        subject="did:jwk:test",
        expires_at=now,
        issuer="did:jwk:issuer",
        acting_on_behalf_of="did:jwk:parent",
        delegation_depth=1,
        claims={"test": True}
    )
    assert resp.valid is True
    assert resp.subject == "did:jwk:test"
    assert resp.delegation_depth == 1
    assert resp.acting_on_behalf_of == "did:jwk:parent"


def test_authorize_request():
    """Test AuthorizeRequest dataclass."""
    req = AuthorizeRequest(
        credential="eyJhbGci...",
        expected_audience="test-api",
        resource="orders",
        action="read",
        want_synthetic_jwt=True
    )
    assert req.credential == "eyJhbGci..."
    assert req.resource == "orders"
    assert req.action == "read"
    assert req.want_synthetic_jwt is True


def test_authorize_response():
    """Test AuthorizeResponse dataclass."""
    agent = GatewayAgentContext(
        acting_on_behalf_of="did:jwk:parent",
        delegation_depth=1,
        scope=["read:orders"]
    )
    resp = AuthorizeResponse(
        allowed=True,
        subject="did:jwk:test",
        synthetic_jwt="eyJhbGci...",
        agent=agent,
        policy_id="policy-1"
    )
    assert resp.allowed is True
    assert resp.synthetic_jwt == "eyJhbGci..."
    assert resp.agent.delegation_depth == 1
    assert resp.policy_id == "policy-1"


def test_delegate_request():
    """Test DelegateRequest dataclass."""
    req = DelegateRequest(
        parent_credential="eyJhbGci...",
        delegate_did="did:jwk:agent",
        scope=["read:orders"],
        ttl_seconds=300,
        claims={"agent_type": "service"}
    )
    assert req.parent_credential == "eyJhbGci..."
    assert req.delegate_did == "did:jwk:agent"
    assert len(req.scope) == 1
    assert req.ttl_seconds == 300
