#!/usr/bin/env python3
"""Setup script to add the issuer to the verifier's trust registry.

This needs to be run once before the basic.py example will work.
"""

import json
import subprocess
import sys

try:
    import httpx
    HTTP_CLIENT = "httpx"
except ImportError:
    try:
        import requests
        HTTP_CLIENT = "requests"
    except ImportError:
        print("Error: Neither httpx nor requests is installed")
        print("Install with: pip install httpx")
        sys.exit(1)


def get_issuer_did():
    """Extract issuer DID from docker logs."""
    print("Extracting issuer DID from docker logs...")

    # Try different container naming conventions
    container_names = [
        "credential-service_issuer_1",  # Underscore (docker-compose v1)
        "credential-service-issuer-1",  # Hyphen (docker compose v2)
    ]

    for container_name in container_names:
        try:
            result = subprocess.run(
                ["docker", "logs", container_name],
                capture_output=True,
                text=True,
                check=False  # Don't raise on non-zero exit
            )

            if result.returncode != 0:
                continue  # Try next container name

            # Combine stdout and stderr (logs can be in either)
            logs = result.stdout + result.stderr

            # Look for issuer_did in JSON logs
            for line in logs.split('\n'):
                # Try to parse as JSON first (structured logging)
                if '"issuer_did"' in line:
                    try:
                        log_data = json.loads(line)
                        if 'issuer_did' in log_data:
                            did = log_data['issuer_did']
                            if did and did.startswith('did:'):
                                print(f"  Found issuer DID: {did[:60]}...")
                                return did
                    except json.JSONDecodeError:
                        pass

                # Fallback: plain text format
                if 'issuer_did=' in line:
                    parts = line.split('issuer_did=')
                    if len(parts) > 1:
                        did = parts[1].split()[0]
                        if did.startswith('did:'):
                            print(f"  Found issuer DID: {did[:60]}...")
                            return did

            # If we got here, we found the container but no DID
            print(f"  ⚠ Found container {container_name} but no issuer DID in logs")
            print("  The issuer may not have issued any credentials yet.")
            return None

        except subprocess.CalledProcessError:
            continue  # Try next container name
        except FileNotFoundError:
            print("  ✗ Docker command not found")
            print("\nMake sure Docker is installed and in your PATH")
            return None

    print("  ✗ Could not find issuer container")
    print("\nTried container names:", ", ".join(container_names))
    print("\nMake sure docker services are running:")
    print("  docker compose up -d")
    return None


def add_to_trust_registry(issuer_did):
    """Add issuer DID to verifier's trust registry."""
    print("\nAdding issuer to trust registry...")

    url = "http://localhost:8081/v1/trust/issuers"
    payload = {"issuer_did": issuer_did}

    try:
        if HTTP_CLIENT == "httpx":
            response = httpx.post(url, json=payload)
        else:
            response = requests.post(url, json=payload)

        if response.status_code in (200, 201):
            print(f"  ✓ Successfully added issuer to trust registry")
            return True
        else:
            print(f"  ✗ Failed to add issuer: HTTP {response.status_code}")
            try:
                error_data = response.json()
                print(f"     Error: {error_data.get('error', 'Unknown error')}")
            except:
                print(f"     Response: {response.text}")
            return False

    except Exception as e:
        print(f"  ✗ Error calling verifier API: {e}")
        print("\nMake sure verifier service is running:")
        print("  curl http://localhost:8081/readyz")
        return False


def verify_setup():
    """Verify the trust registry setup."""
    print("\nVerifying setup...")

    url = "http://localhost:8081/v1/trust/issuers"

    try:
        if HTTP_CLIENT == "httpx":
            response = httpx.get(url)
        else:
            response = requests.get(url)

        if response.status_code == 200:
            data = response.json()
            issuers = data.get('issuers', [])
            print(f"  ✓ Trust registry has {len(issuers)} issuer(s)")
            return True
        else:
            print(f"  ⚠ Could not verify: HTTP {response.status_code}")
            return False

    except Exception as e:
        print(f"  ⚠ Could not verify: {e}")
        return False


def check_services():
    """Check if required services are running."""
    print("Checking services...")

    services_ok = True

    # Check issuer
    try:
        if HTTP_CLIENT == "httpx":
            response = httpx.get("http://localhost:8080/healthz", timeout=2.0)
        else:
            response = requests.get("http://localhost:8080/healthz", timeout=2.0)

        if response.status_code == 200:
            print("  ✓ Issuer service is running (port 8080)")
        else:
            print(f"  ✗ Issuer service returned HTTP {response.status_code}")
            services_ok = False
    except Exception as e:
        print(f"  ✗ Issuer service not reachable: {e}")
        services_ok = False

    # Check verifier
    try:
        if HTTP_CLIENT == "httpx":
            response = httpx.get("http://localhost:8081/readyz", timeout=2.0)
        else:
            response = requests.get("http://localhost:8081/readyz", timeout=2.0)

        if response.status_code == 200:
            print("  ✓ Verifier service is running (port 8081)")
        else:
            print(f"  ✗ Verifier service returned HTTP {response.status_code}")
            services_ok = False
    except Exception as e:
        print(f"  ✗ Verifier service not reachable: {e}")
        services_ok = False

    return services_ok


def main():
    """Main setup flow."""
    print("=" * 70)
    print("Credential Service - Trust Registry Setup")
    print("=" * 70)
    print()

    # Check services
    if not check_services():
        print("\n" + "=" * 70)
        print("⚠️  Services are not running!")
        print("=" * 70)
        print("\nStart services with:")
        print("  cd /Users/pmatern/dev/twilio/credential-service")
        print("  docker compose up -d")
        print("  sleep 3")
        print("\nThen run this script again.")
        return 1

    print()

    # Get issuer DID
    issuer_did = get_issuer_did()
    if not issuer_did:
        print("\n" + "=" * 70)
        print("⚠️  Could not extract issuer DID")
        print("=" * 70)
        print("\nTry manually:")
        print("  docker logs credential-service-issuer-1 2>&1 | grep issuer_did")
        return 1

    # Add to trust registry
    if not add_to_trust_registry(issuer_did):
        print("\n" + "=" * 70)
        print("⚠️  Failed to add issuer to trust registry")
        print("=" * 70)
        return 1

    # Verify
    verify_setup()

    print("\n" + "=" * 70)
    print("✅ Setup complete!")
    print("=" * 70)
    print("\nYou can now run the example:")
    print("  python examples/basic.py")
    print()

    return 0


if __name__ == "__main__":
    sys.exit(main())
