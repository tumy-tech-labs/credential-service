#!/usr/bin/env python3
"""Basic example of using the Credential Service Python SDK.

This example demonstrates the complete flow:
1. Issue a credential
2. Verify the credential
3. Delegate the credential
4. Authorize with the gateway
"""

import sys
from pathlib import Path

# Add parent directory to path for local development
sys.path.insert(0, str(Path(__file__).parent.parent))

from credential_sdk import (
    CredentialServiceClient,
    IssueRequest,
    VerifyRequest,
    AuthorizeRequest,
    DelegateRequest,
    InvalidRequestError,
    UnauthorizedError,
)


def main():
    """Run the basic example flow."""
    print("=== Credential Service Python SDK Example ===\n")

    # Initialize client for local development
    print("Initializing client...")
    with CredentialServiceClient.local_dev() as client:
        try:
            # Step 1: Issue a credential
            print("\n1. Issuing root credential...")
            issued = client.issue_vc(IssueRequest(
                subject_did="did:jwk:eyJrdHkiOiJPS1AiLCJjcnYiOiJFZDI1NTE5IiwieCI6IjExcVlBWUtGMWJuRjNyeEh0Q19FN2I4N1ZRdHJhRUp2WVU0aGRxNFU5SWsifQ",
                ttl_seconds=600,
                claims={
                    "aud": "sample-api",
                    "scope": ["read:orders", "write:orders"],  # Use array format
                    "role": "admin",
                    "department": "engineering"
                }
            ))
            print(f"   ✓ Issued credential")
            print(f"   Credential (first 80 chars): {issued.credential[:80]}...")
            print(f"   Format: {issued.format}")

            # Step 2: Verify the credential
            print("\n2. Verifying credential...")
            verified = client.verify(request=VerifyRequest(
                credential=issued.credential,
                expected_audience="sample-api"
            ))
            print(f"   ✓ Verification successful")
            print(f"   Valid: {verified.valid}")
            print(f"   Subject: {verified.subject}")
            print(f"   Issuer: {verified.issuer}")
            print(f"   Expires at: {verified.expires_at}")
            print(f"   Claims: {verified.claims}")

            # Step 3: Create a delegated credential
            print("\n3. Creating delegated credential...")
            delegated = client.delegate_credential(DelegateRequest(
                parent_credential=issued.credential,
                delegate_did="did:jwk:eyJrdHkiOiJPS1AiLCJjcnYiOiJFZDI1NTE5IiwieCI6InhYeVpBYmNEZWZHaGlKa2xNbm9QcXJTdFV2V3h5WjAxMjM0NTY3ODlBQkMifQ",
                scope=["read:orders"],  # Narrower scope than parent
                ttl_seconds=300,  # Shorter TTL than parent
                claims={"agent_type": "service_account"}
            ))
            print(f"   ✓ Delegated credential created")
            print(f"   Credential (first 80 chars): {delegated.credential[:80]}...")

            # Step 4: Authorize with the gateway
            print("\n4. Authorizing with gateway...")
            decision = client.authorize(AuthorizeRequest(
                credentials=[issued.credential, delegated.credential],
                expected_audience="sample-api",
                resource="orders",
                action="read",
                want_synthetic_jwt=True
            ))
            print(f"   ✓ Authorization decision received")
            print(f"   Allowed: {decision.allowed}")
            print(f"   Subject: {decision.subject}")
            print(f"   Acting on behalf of: {decision.acting_on_behalf_of}")
            print(f"   Delegation depth: {decision.delegation_depth}")
            print(f"   Reason: {decision.reason}")

            if decision.synthetic_jwt:
                print(f"   Synthetic JWT (first 80 chars): {decision.synthetic_jwt[:80]}...")

            if decision.agent:
                print(f"   Agent context:")
                print(f"     - Acting on behalf of: {decision.agent.acting_on_behalf_of}")
                print(f"     - Delegation depth: {decision.agent.delegation_depth}")
                print(f"     - Scope: {decision.agent.scope}")

            # Step 5: Try issuing an SD-JWT
            print("\n5. Issuing SD-JWT credential with selective disclosure...")
            sd_issued = client.issue_sd_jwt(IssueRequest(
                subject_did="did:jwk:eyJrdHkiOiJPS1AiLCJjcnYiOiJFZDI1NTE5IiwieCI6IjExcVlBWUtGMWJuRjNyeEh0Q19FN2I4N1ZRdHJhRUp2WVU0aGRxNFU5SWsifQ",
                ttl_seconds=600,
                claims={
                    "email": "alice@example.com",
                    "department": "engineering",
                    "clearance_level": "secret",
                    "scope": "read:orders"
                }
            ))
            print(f"   ✓ SD-JWT issued")
            print(f"   Credential (first 80 chars): {sd_issued.credential[:80]}...")
            print(f"   Number of disclosures: {len(sd_issued.disclosures or [])}")
            if sd_issued.disclosures:
                print(f"   First disclosure: {sd_issued.disclosures[0][:80]}...")

            print("\n=== Example completed successfully! ===")

        except InvalidRequestError as e:
            print(f"\n✗ Invalid request: {e.message}")
            print(f"  Status code: {e.status_code}")
            sys.exit(1)
        except UnauthorizedError as e:
            print(f"\n✗ Unauthorized: {e.message}")
            print(f"  Status code: {e.status_code}")
            sys.exit(1)
        except Exception as e:
            print(f"\n✗ Unexpected error: {e}")
            import traceback
            traceback.print_exc()
            sys.exit(1)


if __name__ == "__main__":
    main()
