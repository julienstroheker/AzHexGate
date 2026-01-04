# Testing Guide for Azure Relay Hybrid Connection Creation

This guide explains how to test the new Hybrid Connection creation functionality.

## Prerequisites

1. Azure CLI installed and authenticated: `az login`
2. Azure Relay namespace deployed
3. Proper Azure RBAC permissions (the App Service Managed Identity has "Azure Relay Owner" role via Bicep)

## Required Environment Variables

Set the following environment variables before running the gateway:

```bash
# Azure Relay configuration
export AZHEXGATE_RELAY_NAMESPACE="azhexgate-relay-dev"  # Your relay namespace name
export AZHEXGATE_RELAY_KEY_NAME="RootManageSharedAccessKey"
export AZHEXGATE_RELAY_KEY="<your-relay-key>"  # Get from Azure Portal

# Azure credentials for Hybrid Connection creation
export AZURE_SUBSCRIPTION_ID="<your-subscription-id>"
export AZURE_RESOURCE_GROUP="<your-resource-group>"  # Resource group containing the relay

# Optional: Base domain
export AZHEXGATE_BASE_DOMAIN="azhexgate.com"
```

### Getting Azure Relay Credentials

```bash
# Get your subscription ID
az account show --query id -o tsv

# Get the Relay shared access key
az relay namespace authorization-rule keys list \
  --resource-group <your-resource-group> \
  --namespace-name <your-relay-namespace> \
  --name RootManageSharedAccessKey \
  --query primaryKey -o tsv
```

## Testing Steps

### 1. Start the Gateway

```bash
go run gateway/main.go start -v
```

You should see:
```
INFO Management service initialized with real Azure Relay integration
INFO Gateway listening port=8080
```

### 2. Create a Tunnel

```bash
curl -X POST -d '{"local_port":3000}' http://localhost:8080/api/tunnels -v
```

Expected response:
```json
{
  "public_url": "https://abc12345.azhexgate.com",
  "relay_endpoint": "azhexgate-relay-dev.servicebus.windows.net",
  "hybrid_connection_name": "hc-abc12345",
  "listener_token": "SharedAccessSignature sr=...",
  "session_id": "..."
}
```

### 3. Verify Hybrid Connection Created

Check in Azure Portal or via CLI:

```bash
az relay hyco list \
  --resource-group <your-resource-group> \
  --namespace-name <your-relay-namespace> \
  --query "[].name" -o table
```

You should see the newly created Hybrid Connection (e.g., `hc-abc12345`).

### 4. Start the Client

```bash
go run client/main.go start -v
```

The client should now successfully connect without the 404 error:

Expected output:
```
INFO Tunnel created, preparing to start listener
INFO Initializing Azure Relay listener
INFO Listener loop started, waiting for connections...
INFO Starting listener loop
```

**No more 404 errors!**

## Authentication Methods

The gateway uses Azure Default Credential, which tries authentication in this order:

1. **Environment Variables** (`AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`)
2. **Managed Identity** (when running in App Service)
3. **Azure CLI** (for local development - `az login`)

For local testing, just use `az login` and you're good to go!

## Troubleshooting

### Error: "failed to create relay manager"

**Cause:** Missing `AZURE_SUBSCRIPTION_ID` or `AZURE_RESOURCE_GROUP`

**Solution:** Set both environment variables.

### Error: "failed to create hybrid connection: ..."

**Cause:** Insufficient permissions or authentication issues

**Solution:** 
- Verify you're logged in: `az account show`
- Check you have the right subscription: `az account set --subscription <id>`
- Ensure you have Contributor or Owner role on the resource group

### Still Getting 404 from Client

**Cause:** Hybrid Connection wasn't created (check gateway logs)

**Solution:** 
- Verify gateway logs show "Management service initialized with real Azure Relay integration"
- Check Azure Portal to confirm the Hybrid Connection exists
- Try creating another tunnel

## Cleanup

To delete old Hybrid Connections:

```bash
# List all
az relay hyco list \
  --resource-group <your-resource-group> \
  --namespace-name <your-relay-namespace> \
  --query "[].name" -o table

# Delete one
az relay hyco delete \
  --resource-group <your-resource-group> \
  --namespace-name <your-relay-namespace> \
  --name hc-abc12345
```

## Production Deployment

When deploying to App Service:

1. The App Service uses Managed Identity (configured via Bicep)
2. No need to set credential environment variables
3. Only set:
   - `AZHEXGATE_RELAY_NAMESPACE`
   - `AZHEXGATE_RELAY_KEY_NAME`
   - `AZHEXGATE_RELAY_KEY`
   - `AZURE_SUBSCRIPTION_ID`
   - `AZURE_RESOURCE_GROUP`

The Managed Identity automatically has the "Azure Relay Owner" role assigned via the `relay-rbac.bicep` module.
