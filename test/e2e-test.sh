#!/bin/bash
# End-to-end test script for AzHexGate local testing

# Configuration
SUBDOMAIN="c12aaac4"
BASE_DOMAIN="azhexgate.com"
FULL_DOMAIN="${SUBDOMAIN}.${BASE_DOMAIN}"
GATEWAY_PORT=8080
LOCAL_APP_PORT=3000

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== AzHexGate Local E2E Test ===${NC}\n"

# Step 1: Check if hosts file has the entry
echo -e "${YELLOW}Step 1: Checking hosts file...${NC}"
if grep -q "127.0.0.1.*${FULL_DOMAIN}" /etc/hosts; then
    echo -e "${GREEN}✓ Found ${FULL_DOMAIN} in /etc/hosts${NC}"
else
    echo -e "${RED}✗ Missing hosts entry${NC}"
    echo -e "Add this line to /etc/hosts:"
    echo -e "  ${YELLOW}127.0.0.1 ${FULL_DOMAIN}${NC}"
    echo -e "\nRun:"
    echo -e "  ${YELLOW}echo '127.0.0.1 ${FULL_DOMAIN}' | sudo tee -a /etc/hosts${NC}"
    exit 1
fi

# Step 2: Check if local app is running on port 3000
echo -e "\n${YELLOW}Step 2: Checking local application on port ${LOCAL_APP_PORT}...${NC}"
if curl -s -o /dev/null -w "%{http_code}" http://localhost:${LOCAL_APP_PORT}/ > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Local application responding on port ${LOCAL_APP_PORT}${NC}"
else
    echo -e "${RED}✗ No application running on port ${LOCAL_APP_PORT}${NC}"
    echo -e "Start a local server, e.g.:"
    echo -e "  ${YELLOW}python3 -m http.server ${LOCAL_APP_PORT}${NC}"
    exit 1
fi

# Step 3: Check if gateway is running
echo -e "\n${YELLOW}Step 3: Checking gateway on port ${GATEWAY_PORT}...${NC}"
if curl -s -o /dev/null http://localhost:${GATEWAY_PORT}/healthz; then
    echo -e "${GREEN}✓ Gateway responding on port ${GATEWAY_PORT}${NC}"
else
    echo -e "${RED}✗ Gateway not running on port ${GATEWAY_PORT}${NC}"
    echo -e "Start the gateway with environment variables:"
    echo -e "  ${YELLOW}AZHEXGATE_RELAY_NAMESPACE=azhexgate-relay-dev \\${NC}"
    echo -e "  ${YELLOW}AZHEXGATE_RELAY_KEY_NAME=RootManageSharedAccessKey \\${NC}"
    echo -e "  ${YELLOW}AZHEXGATE_RELAY_KEY=<your-key> \\${NC}"
    echo -e "  ${YELLOW}AZHEXGATE_BASE_DOMAIN=azhexgate.com \\${NC}"
    echo -e "  ${YELLOW}./bin/gateway start${NC}"
    exit 1
fi

# Step 4: Check if client is running
echo -e "\n${YELLOW}Step 4: Checking client connection...${NC}"
echo -e "${YELLOW}(Make sure you've run: go run client/main.go start --port ${LOCAL_APP_PORT})${NC}"
read -p "Is the client running and connected? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
fi

# Step 5: Make test request
echo -e "\n${YELLOW}Step 5: Making test request through tunnel...${NC}"
echo -e "Requesting: ${YELLOW}http://${FULL_DOMAIN}:${GATEWAY_PORT}/${NC}"

RESPONSE=$(curl -v -H "Host: ${FULL_DOMAIN}" http://127.0.0.1:${GATEWAY_PORT}/ 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | grep "< HTTP" | awk '{print $3}')

if [[ -n "$HTTP_CODE" ]]; then
    if [[ "$HTTP_CODE" == "200" ]]; then
        echo -e "${GREEN}✓ Success! Received HTTP ${HTTP_CODE}${NC}"
        echo -e "\n${GREEN}=== Test passed! ===${NC}"
    else
        echo -e "${YELLOW}⚠ Received HTTP ${HTTP_CODE}${NC}"
        echo -e "\nFull response:"
        echo "$RESPONSE"
    fi
else
    echo -e "${RED}✗ No response received${NC}"
    echo -e "\nFull output:"
    echo "$RESPONSE"
    exit 1
fi
