#!/bin/sh
set -e

# Validate required environment variables
check_required_env() {
    local var_name="$1"
    local var_value
    eval var_value="\$$var_name"
    if [ -z "$var_value" ]; then
        echo "ERROR: Required environment variable $var_name is not set"
        exit 1
    fi
}

echo "CalBridgeSync starting..."

# Check required environment variables
check_required_env "BASE_URL"
check_required_env "OIDC_ISSUER"
check_required_env "OIDC_CLIENT_ID"
check_required_env "OIDC_CLIENT_SECRET"
check_required_env "OIDC_REDIRECT_URL"
check_required_env "ENCRYPTION_KEY"
check_required_env "SESSION_SECRET"
check_required_env "DEFAULT_DEST_URL"

# Validate encryption key length (should be 64 hex chars = 32 bytes)
if [ ${#ENCRYPTION_KEY} -ne 64 ]; then
    echo "ERROR: ENCRYPTION_KEY must be exactly 64 hexadecimal characters (32 bytes)"
    exit 1
fi

# Validate session secret length (minimum 32 chars)
if [ ${#SESSION_SECRET} -lt 32 ]; then
    echo "ERROR: SESSION_SECRET must be at least 32 characters"
    exit 1
fi

# Ensure data directory exists and is writable
if [ ! -d "/app/data" ]; then
    echo "ERROR: Data directory /app/data does not exist"
    exit 1
fi

if [ ! -w "/app/data" ]; then
    echo "ERROR: Data directory /app/data is not writable"
    exit 1
fi

echo "Environment validation complete"
echo "Starting CalBridgeSync on port ${PORT:-8080}..."

# Execute the main command
exec "$@"
