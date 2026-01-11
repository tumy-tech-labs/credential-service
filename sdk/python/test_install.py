#!/usr/bin/env python3
"""Quick test to verify SDK installation."""

import sys


def test_imports():
    """Test that all modules can be imported."""
    print("Testing imports...")

    try:
        from credential_sdk import (
            CredentialServiceClient,
            IssueRequest,
            IssueResponse,
            VerifyRequest,
            VerifyResponse,
            AuthorizeRequest,
            AuthorizeResponse,
            DelegateRequest,
            DelegateResponse,
            GatewayAgentContext,
            CredentialServiceError,
            NetworkError,
            InvalidRequestError,
            UnauthorizedError,
            ServerError,
        )
        print("  ✓ All imports successful")
        return True
    except ImportError as e:
        print(f"  ✗ Import error: {e}")
        return False


def test_client_creation():
    """Test client can be created."""
    print("\nTesting client creation...")

    try:
        from credential_sdk import CredentialServiceClient

        # Test different creation methods
        client1 = CredentialServiceClient.local_dev()
        print(f"  ✓ Local dev client: {client1.issuer_url}")

        client2 = CredentialServiceClient(
            issuer_url="http://test:8080",
            verifier_url="http://test:8081"
        )
        print(f"  ✓ Custom client: {client2.issuer_url}")

        return True
    except Exception as e:
        print(f"  ✗ Client creation error: {e}")
        return False


def test_http_backend():
    """Test that an HTTP backend is available."""
    print("\nTesting HTTP backend availability...")

    try:
        import httpx
        print("  ✓ httpx is available")
        return True
    except ImportError:
        pass

    try:
        import requests
        print("  ✓ requests is available")
        return True
    except ImportError:
        pass

    print("  ✗ No HTTP backend (httpx or requests) found")
    print("    Install with: pip install httpx  OR  pip install requests")
    return False


def test_types():
    """Test that type definitions work."""
    print("\nTesting type definitions...")

    try:
        from credential_sdk import IssueRequest, AuthorizeRequest

        req1 = IssueRequest(
            subject_did="did:jwk:test",
            ttl_seconds=600,
            claims={"test": True}
        )
        print(f"  ✓ IssueRequest created: {req1.subject_did}")

        req2 = AuthorizeRequest(
            credential="test",
            expected_audience="test-api",
            want_synthetic_jwt=True
        )
        print(f"  ✓ AuthorizeRequest created: want_jwt={req2.want_synthetic_jwt}")

        return True
    except Exception as e:
        print(f"  ✗ Type definition error: {e}")
        return False


def main():
    """Run all tests."""
    print("=" * 60)
    print("Credential Service Python SDK - Installation Test")
    print("=" * 60)

    results = []
    results.append(("Imports", test_imports()))
    results.append(("Client Creation", test_client_creation()))
    results.append(("HTTP Backend", test_http_backend()))
    results.append(("Type Definitions", test_types()))

    print("\n" + "=" * 60)
    print("Test Results:")
    print("=" * 60)

    all_passed = True
    for name, passed in results:
        status = "✓ PASS" if passed else "✗ FAIL"
        print(f"  {status}: {name}")
        if not passed:
            all_passed = False

    print("=" * 60)

    if all_passed:
        print("\n🎉 All tests passed! SDK is properly installed.")
        print("\nNext steps:")
        print("  1. Start services: docker compose up -d")
        print("  2. Run example: python examples/basic.py")
        print("  3. Run tests: pytest")
        return 0
    else:
        print("\n⚠️  Some tests failed. Please check the errors above.")
        print("\nCommon fixes:")
        print("  • Install HTTP backend: pip install httpx")
        print("  • Reinstall SDK: pip install -e '.[httpx,dev]'")
        print("  • See INSTALL.md for troubleshooting")
        return 1


if __name__ == "__main__":
    sys.exit(main())
