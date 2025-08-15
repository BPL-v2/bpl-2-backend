#!/bin/bash

# Function to show usage
show_usage() {
    echo "Usage: $0 -id=<user_id> [-permissions=<permission1,permission2,...>]"
    echo ""
    echo "Options:"
    echo "  -id=<number>              User ID (required)"
    echo "  -permissions=<list>       Comma-separated list of permissions (optional)"
    echo ""
    echo "Available permissions: admin, objective_designer, submission_judge, manager"
    echo ""
    echo "Examples:"
    echo "  $0 -id=1"
    echo "  $0 -id=1 -permissions=admin,manager"
    echo "  $0 -id=5 -permissions=admin,objective_designer,submission_judge,manager"
    exit 1
}

# Initialize variables
USER_ID=""
PERMISSIONS=""

# Parse command line arguments
for arg in "$@"; do
    case $arg in
        -id=*)
            USER_ID="${arg#*=}"
            ;;
        -permissions=*)
            PERMISSIONS="${arg#*=}"
            ;;
        -h|--help)
            show_usage
            ;;
        *)
            echo "Error: Unknown argument $arg"
            show_usage
            ;;
    esac
done

# Validate required arguments
if [ -z "$USER_ID" ]; then
    echo "Error: User ID is required"
    show_usage
fi

# Validate user ID is a number
if ! [[ "$USER_ID" =~ ^[0-9]+$ ]]; then
    echo "Error: User ID must be a number"
    exit 1
fi

# Function to load .env file safely
load_env() {
    if [ -f ".env" ]; then
        # Read .env file line by line and export variables
        while IFS='=' read -r key value; do
            # Skip comments and empty lines
            if [[ $key =~ ^[[:space:]]*# ]] || [[ -z $key ]]; then
                continue
            fi
            # Remove leading/trailing whitespace
            key=$(echo "$key" | xargs)
            value=$(echo "$value" | xargs)
            # Export the variable
            if [[ -n $key && -n $value ]]; then
                export "$key"="$value"
            fi
        done < .env
    else
        echo "Warning: .env file not found in current directory"
    fi
}

# Load environment variables from .env
load_env

# Check if JWT_SECRET environment variable is set
if [ -z "$JWT_SECRET" ]; then
    echo "Error: JWT_SECRET environment variable is not set"
    echo "Make sure JWT_SECRET is defined in your .env file or environment"
    exit 1
fi

# Base64url encode function (removes padding and replaces characters)
base64url_encode() {
    echo -n "$1" | base64 | tr '+/' '-_' | tr -d '='
}

# Calculate expiration time (1 year from now)
exp=$(date -d "+100 year" +%s)

# Create JWT header
header='{"alg":"HS256","typ":"JWT"}'

# Create JWT payload with dynamic values
if [ -n "$PERMISSIONS" ]; then
    # Convert comma-separated permissions to JSON array
    IFS=',' read -ra PERM_ARRAY <<< "$PERMISSIONS"
    permissions_json="["
    for i in "${!PERM_ARRAY[@]}"; do
        if [ $i -gt 0 ]; then
            permissions_json+=","
        fi
        permissions_json+="\"${PERM_ARRAY[$i]}\""
    done
    permissions_json+="]"
else
    # Empty permissions array
    permissions_json="[]"
fi

payload="{\"exp\":$exp,\"permissions\":$permissions_json,\"user_id\":$USER_ID}"

# Base64url encode header and payload
encoded_header=$(base64url_encode "$header")
encoded_payload=$(base64url_encode "$payload")

# Create the signature input (header.payload)
signature_input="${encoded_header}.${encoded_payload}"

# Create HMAC-SHA256 signature
signature=$(echo -n "$signature_input" | openssl dgst -sha256 -hmac "$JWT_SECRET" -binary | base64 | tr '+/' '-_' | tr -d '=')

# Construct the final JWT
jwt="${signature_input}.${signature}"

echo "$jwt"