"""Type definitions for the Credential Service SDK."""

from dataclasses import dataclass, field
from datetime import datetime
from typing import Any, Dict, List, Optional


@dataclass
class IssueRequest:
    """Request to issue a verifiable credential."""

    subject_did: str
    ttl_seconds: int = 600
    claims: Optional[Dict[str, Any]] = None
    format: Optional[str] = None


@dataclass
class IssueResponse:
    """Response containing an issued credential."""

    credential: str
    disclosures: Optional[List[str]] = None
    format: Optional[str] = None
    api_version: Optional[str] = None


@dataclass
class VerifyRequest:
    """Request to verify a credential or credential chain."""

    credential: Optional[str] = None
    credentials: Optional[List[str]] = None
    expected_audience: Optional[str] = None
    disclosures: Optional[List[str]] = None
    format: Optional[str] = None


@dataclass
class VerifyResponse:
    """Response containing verification details."""

    valid: bool
    subject: str
    expires_at: datetime
    issuer: Optional[str] = None
    acting_on_behalf_of: Optional[str] = None
    delegation_depth: int = 0
    claims: Optional[Dict[str, Any]] = None
    synthetic_jwt: Optional[str] = None
    api_version: Optional[str] = None


@dataclass
class AuthorizeRequest:
    """Request for gateway authorization decision."""

    credential: Optional[str] = None
    credentials: Optional[List[str]] = None
    expected_audience: Optional[str] = None
    want_synthetic_jwt: bool = False
    resource: Optional[str] = None
    action: Optional[str] = None


@dataclass
class GatewayAgentContext:
    """Agent metadata when delegation is detected."""

    acting_on_behalf_of: str
    delegation_depth: int
    scope: Optional[List[str]] = None


@dataclass
class AuthorizeResponse:
    """Response from gateway authorization."""

    allowed: bool
    subject: Optional[str] = None
    acting_on_behalf_of: Optional[str] = None
    delegation_depth: int = 0
    claims: Optional[Dict[str, Any]] = None
    reason: Optional[str] = None
    synthetic_jwt: Optional[str] = None
    agent: Optional[GatewayAgentContext] = None
    api_version: Optional[str] = None
    policy_id: Optional[str] = None
    tenant_id: Optional[str] = None  # Added for multi-tenancy support


@dataclass
class DelegateRequest:
    """Request to delegate a credential."""

    parent_credential: str
    delegate_did: str
    scope: List[str]
    ttl_seconds: int
    claims: Optional[Dict[str, Any]] = None


@dataclass
class DelegateResponse:
    """Response containing a delegated credential."""

    credential: str
    api_version: Optional[str] = None
