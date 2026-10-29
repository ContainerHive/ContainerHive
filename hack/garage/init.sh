#!/bin/bash
set -e

GARAGE_ADMIN="http://garage:3903"
BUCKET_NAME="buildkit-cache"
KEY_NAME="buildkit-key"
# Fixed credentials matching building.py (must be valid hex: GK+24 hex chars, 64 hex chars)
ACCESS_KEY_ID="GK31337cafe000000000000000"
SECRET_KEY="1337cafe0000000000000000000000000000000000000000000000000000dead"
AUTH_HEADER="Authorization: Bearer admin-token-for-local-dev"

echo "Waiting for Garage admin API..."
until curl -sf -H "$AUTH_HEADER" "${GARAGE_ADMIN}/v2/GetClusterStatus" > /dev/null 2>&1; do
    sleep 1
done
echo "Garage is ready"

# Get cluster status and node ID
echo "Getting cluster status..."
CLUSTER_STATUS=$(curl -sf -H "$AUTH_HEADER" "${GARAGE_ADMIN}/v2/GetClusterStatus")
NODE_ID=$(echo "$CLUSTER_STATUS" | jq -r '.nodes[0].id')

# Check if layout already configured
LAYOUT=$(curl -sf -H "$AUTH_HEADER" "${GARAGE_ADMIN}/v2/GetClusterLayout")
STAGED_COUNT=$(echo "$LAYOUT" | jq '.stagedRoleChanges | length')

if [[ "$STAGED_COUNT" -eq 0 ]]; then
    ROLES=$(echo "$LAYOUT" | jq '.roles | length')
    if [[ "$ROLES" -eq 0 ]]; then
        echo "Configuring node layout..."
        # v2 API uses roles array format
        curl -sf -X POST -H "$AUTH_HEADER" \
            -H "Content-Type: application/json" \
            -d "{\"roles\": [{\"id\": \"$NODE_ID\", \"zone\": \"dc1\", \"capacity\": 1073741824, \"tags\": []}]}" \
            "${GARAGE_ADMIN}/v2/UpdateClusterLayout"

        echo "Applying layout..."
        CURRENT_VERSION=$(curl -sf -H "$AUTH_HEADER" "${GARAGE_ADMIN}/v2/GetClusterLayout" | jq '.version')
        curl -sf -X POST -H "$AUTH_HEADER" \
            -H "Content-Type: application/json" \
            -d "{\"version\": $((CURRENT_VERSION + 1))}" \
            "${GARAGE_ADMIN}/v2/ApplyClusterLayout"
    fi
fi

# Check if bucket exists
echo "Checking bucket..."
BUCKETS=$(curl -sf -H "$AUTH_HEADER" "${GARAGE_ADMIN}/v2/ListBuckets")
BUCKET_EXISTS=$(echo "$BUCKETS" | jq -r ".[] | select(.globalAliases[]? == \"$BUCKET_NAME\") | .id")

if [[ -z "$BUCKET_EXISTS" ]]; then
    echo "Creating bucket ${BUCKET_NAME}..."
    BUCKET_RESULT=$(curl -sf -X POST -H "$AUTH_HEADER" \
        -H "Content-Type: application/json" \
        -d "{\"globalAlias\": \"$BUCKET_NAME\"}" \
        "${GARAGE_ADMIN}/v2/CreateBucket")
    BUCKET_ID=$(echo "$BUCKET_RESULT" | jq -r '.id')
else
    BUCKET_ID="$BUCKET_EXISTS"
fi
echo "Bucket ID: $BUCKET_ID"

# Check if key exists
echo "Checking access key..."
KEYS=$(curl -sf -H "$AUTH_HEADER" "${GARAGE_ADMIN}/v2/ListKeys")
KEY_EXISTS=$(echo "$KEYS" | jq -r ".[] | select(.name == \"$KEY_NAME\") | .id")

if [[ -z "$KEY_EXISTS" ]]; then
    echo "Creating access key ${KEY_NAME}..."
    curl -sf -X POST -H "$AUTH_HEADER" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"$KEY_NAME\", \"accessKeyId\": \"$ACCESS_KEY_ID\", \"secretAccessKey\": \"$SECRET_KEY\"}" \
        "${GARAGE_ADMIN}/v2/ImportKey"
fi

# Grant bucket permissions to key
echo "Granting bucket permissions..."
curl -sf -X POST -H "$AUTH_HEADER" \
    -H "Content-Type: application/json" \
    -d "{\"bucketId\": \"$BUCKET_ID\", \"accessKeyId\": \"$ACCESS_KEY_ID\", \"permissions\": {\"read\": true, \"write\": true, \"owner\": true}}" \
    "${GARAGE_ADMIN}/v2/AllowBucketKey"

echo "Garage initialization complete!"
echo "  Bucket: $BUCKET_NAME"
echo "  Access Key ID: $ACCESS_KEY_ID"