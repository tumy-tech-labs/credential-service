# Installation Guide

## The Issue You Encountered

The error occurred because:
1. The virtual environment was created without pip properly bootstrapped
2. The old `python setup.py develop` command is deprecated in favor of modern `pip install -e .`

## Fresh Installation (Recommended)

Follow these steps for a clean installation:

### Step 1: Remove Old Virtual Environment
```bash
cd /Users/pmatern/dev/twilio/credential-service/sdk/python
rm -rf example/  # or whatever your venv directory is named
```

### Step 2: Create Fresh Virtual Environment
```bash
# Use Python 3.8 or later
python3 -m venv venv

# Activate it
source venv/bin/activate  # On macOS/Linux
# OR
venv\Scripts\activate  # On Windows
```

### Step 3: Upgrade pip (Important!)
```bash
pip install --upgrade pip setuptools wheel
```

### Step 4: Install the SDK in Editable Mode
```bash
# Option 1: Install with httpx (recommended)
pip install -e ".[httpx,dev]"

# Option 2: Install with requests
pip install -e ".[requests,dev]"

# Option 3: Install both
pip install -e ".[httpx,requests,dev]"
```

## Verify Installation

```bash
# Check the package is installed
pip list | grep credential-service

# Run tests
pytest

# Try importing
python -c "from credential_sdk import CredentialServiceClient; print('Success!')"
```

## Alternative: Install Without Editable Mode

If you just want to use the SDK without development:

```bash
pip install httpx  # or: pip install requests
python -m pip install /Users/pmatern/dev/twilio/credential-service/sdk/python
```

## Running the Example

```bash
# Make sure services are running first
cd /Users/pmatern/dev/twilio/credential-service
docker compose up -d

# Wait a few seconds
sleep 3

# Run the example
cd sdk/python
source venv/bin/activate
python examples/basic.py
```

## Troubleshooting

### Problem: "No module named pip" in venv

**Solution**: The venv wasn't created properly. Delete and recreate:
```bash
rm -rf venv
python3 -m venv venv
source venv/bin/activate
pip install --upgrade pip
```

### Problem: "SetuptoolsDeprecationWarning" about license

**Solution**: This is just a warning and won't prevent installation. The license format has been fixed in the latest pyproject.toml.

### Problem: Import errors when running examples

**Solution**: Make sure you've installed with one of the HTTP backends:
```bash
pip install httpx  # or: pip install requests
```

### Problem: Tests fail with "ModuleNotFoundError"

**Solution**: Install with dev dependencies:
```bash
pip install -e ".[httpx,dev]"
```

## Development Workflow

Once installed, you can:

```bash
# Make changes to code
vim credential_sdk/client.py

# Changes are immediately available (editable install)
python -c "from credential_sdk import CredentialServiceClient"

# Run tests
pytest

# Run specific test
pytest tests/test_client.py::test_issue_vc -v
```

## Clean Installation Command Summary

For quick copy-paste:

```bash
cd /Users/pmatern/dev/twilio/credential-service/sdk/python
rm -rf venv
python3 -m venv venv
source venv/bin/activate
pip install --upgrade pip setuptools wheel
pip install -e ".[httpx,dev]"
pytest
python examples/basic.py
```
