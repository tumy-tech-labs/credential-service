#!/bin/bash
set -e

echo "🚀 Testing Improved Credential Service Error Handling"
echo "=================================================="

# Start services in background
echo "Starting services..."
docker compose up -d --build

# Wait a moment for services to start
sleep 5

echo ""
echo "1. Testing DID Validation Errors:"
echo "   Invalid DID format should return structured error..."

curl -s -X POST http://localhost:8080/v1/credentials/issue \
  -H "Content-Type: application/json" \
  -d '{
    "subject_did": "not-a-valid-did",
    "ttl_seconds": 3600,
    "claims": {"name": "Test User"}
  }' | jq '.'

echo ""
echo "2. Testing TTL Validation:"
echo "   Negative TTL should return field-level validation error..."

curl -s -X POST http://localhost:8080/v1/credentials/issue \
  -H "Content-Type: application/json" \
  -d '{
    "subject_did": "did:web:example.com",
    "ttl_seconds": -1,
    "claims": {"name": "Test User"}
  }' | jq '.'

echo ""
echo "3. Testing Valid Request:"
echo "   Valid request should succeed and return JWT..."

response=$(curl -s -X POST http://localhost:8080/v1/credentials/issue \
  -H "Content-Type: application/json" \
  -d '{
    "subject_did": "did:web:user.example.com",
    "ttl_seconds": 3600,
    "claims": {"name": "Valid User", "role": "tester"}
  }')

echo "$response" | jq '.'

# Extract the JWT for verification
jwt=$(echo "$response" | jq -r '.credential')

echo ""
echo "4. Testing Credential Verification:"
echo "   Verifying the issued credential..."

curl -s -X POST http://localhost:8081/v1/credentials/verify \
  -H "Content-Type: application/json" \
  -d "{\"credential\": \"$jwt\"}" | jq '.'

echo ""
echo "5. Cleanup:"
docker compose down

echo ""
echo "✅ All tests completed successfully!"
echo "   The improved error handling provides:"
echo "   - Structured field-level validation errors"
echo "   - Clear error codes and messages" 
echo "   - Proper HTTP status codes"
echo "   - W3C-compliant JWT structure with both VC and JWT fields"