# AzHexGate Infrastructure

This document describes the Azure infrastructure components for AzHexGate and provides instructions for deployment using Azure CLI and Bicep.

## Overview

AzHexGate's infrastructure is defined as code using Bicep templates. The infrastructure follows a modular design with reusable components that can be deployed to any Azure subscription.

## Architecture Components

### 1. Azure Relay Namespace

**Purpose:** Provides secure, bidirectional communication between the Cloud Gateway and Local Clients through Hybrid Connections.

**Details:**
- **Resource Type:** `Microsoft.Relay/namespaces`
- **SKU:** Standard
- **Module:** `infra/modules/relay.bicep`

**Key Features:**
- Enables outbound-only connections from local clients (no inbound firewall rules needed)
- Acts as rendezvous point between Gateway (Sender) and Client (Listener)
- Supports hybrid connection-based tunneling

**Outputs:**
- Relay namespace name
- Relay namespace endpoint (Service Bus endpoint)
- Resource ID

### 2. App Service Plan

**Purpose:** Hosts the compute resources for the Cloud Gateway application.

**Details:**
- **Resource Type:** `Microsoft.Web/serverfarms`
- **SKU:** B1 (Basic)
- **OS:** Linux
- **Module:** `infra/modules/appservice.bicep`

**Configuration:**
- Reserved for Linux containers
- Can be scaled up/out based on traffic requirements

### 3. App Service

**Purpose:** Runs the AzHexGate Cloud Gateway application that receives public HTTP(S) traffic and forwards it through Azure Relay.

**Details:**
- **Resource Type:** `Microsoft.Web/sites`
- **Runtime:** Go 1.23 (Linux)
- **Module:** `infra/modules/appservice.bicep`

**Key Features:**
- **System-Assigned Managed Identity:** Enables passwordless authentication to Azure services (Relay, Key Vault, etc.)
- **HTTPS Only:** Forces secure connections
- **Always On:** Keeps the application loaded
- **HTTP/2 Enabled:** Supports modern HTTP protocol
- **TLS 1.2 Minimum:** Ensures secure communications
- **FTPS Disabled:** Reduces attack surface

**Outputs:**
- App Service name
- Default hostname
- Managed Identity Principal ID and Tenant ID
- Resource ID

## Resource Naming Convention

Resources follow a consistent naming pattern:

```
{baseName}-{resourceType}-{environmentName}
```

**Examples:**
- Relay Namespace: `azhexgate-relay-dev`
- App Service Plan: `azhexgate-plan-dev`
- App Service: `azhexgate-app-dev`

## Tagging Strategy

All resources are tagged with:
- `environment`: Environment name (e.g., dev, prod)
- `project`: "AzHexGate"
- `managedBy`: "Bicep"

Additional tags can be added via the `tags` parameter.

## Deployment Instructions

### Prerequisites

1. **Azure Subscription:** An active Azure subscription
2. **Azure CLI:** Install from [https://docs.microsoft.com/cli/azure/install-azure-cli](https://docs.microsoft.com/cli/azure/install-azure-cli)
3. **Bicep CLI:** Installed automatically with Azure CLI 2.20.0 or later
4. **Permissions:** Contributor role on the target subscription or resource group

### Step 1: Login to Azure

```bash
az login
```

Select your subscription:

```bash
az account set --subscription "<subscription-id-or-name>"
```

### Step 2: Create a Resource Group

```bash
az group create \
  --name "azhexgate-rg-dev" \
  --location "eastus"
```

**Note:** Choose a location close to your users or services.

### Step 3: Deploy the Infrastructure

#### Option A: Deploy with Default Parameters

```bash
az deployment group create \
  --resource-group "azhexgate-rg-dev" \
  --template-file infra/main.bicep
```

This will deploy with default values:
- `environmentName`: "dev"
- `baseName`: "azhexgate"
- `location`: Resource group location

#### Option B: Deploy with Custom Parameters

```bash
az deployment group create \
  --resource-group "azhexgate-rg-dev" \
  --template-file infra/main.bicep \
  --parameters environmentName=prod \
  --parameters baseName=myapp \
  --parameters location=westus2
```

#### Option C: Deploy with Parameter File

Create a parameters file `infra/main.parameters.json`:

```json
{
  "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentParameters.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "environmentName": {
      "value": "prod"
    },
    "baseName": {
      "value": "azhexgate"
    },
    "location": {
      "value": "eastus"
    },
    "tags": {
      "value": {
        "environment": "prod",
        "project": "AzHexGate",
        "managedBy": "Bicep",
        "costCenter": "Engineering"
      }
    }
  }
}
```

Deploy with parameters file:

```bash
az deployment group create \
  --resource-group "azhexgate-rg-prod" \
  --template-file infra/main.bicep \
  --parameters infra/main.parameters.json
```

### Step 4: Verify Deployment

Check deployment status:

```bash
az deployment group show \
  --resource-group "azhexgate-rg-dev" \
  --name "main"
```

List deployed resources:

```bash
az resource list \
  --resource-group "azhexgate-rg-dev" \
  --output table
```

### Step 5: Retrieve Outputs

Get deployment outputs (useful for configuring the application):

```bash
az deployment group show \
  --resource-group "azhexgate-rg-dev" \
  --name "main" \
  --query properties.outputs
```

Example output:

```json
{
  "appServiceHostName": {
    "type": "String",
    "value": "azhexgate-app-dev.azurewebsites.net"
  },
  "appServiceName": {
    "type": "String",
    "value": "azhexgate-app-dev"
  },
  "appServicePrincipalId": {
    "type": "String",
    "value": "12345678-1234-1234-1234-123456789abc"
  },
  "relayNamespaceEndpoint": {
    "type": "String",
    "value": "https://azhexgate-relay-dev.servicebus.windows.net:443/"
  },
  "relayNamespaceName": {
    "type": "String",
    "value": "azhexgate-relay-dev"
  }
}
```

## Validation and Testing

### Validate Templates Before Deployment

Run Bicep validation to check for syntax errors:

```bash
# Validate main template
az bicep build --file infra/main.bicep

# Validate individual modules
az bicep build --file infra/modules/relay.bicep
az bicep build --file infra/modules/appservice.bicep
```

### What-If Deployment

Preview changes before actual deployment:

```bash
az deployment group what-if \
  --resource-group "azhexgate-rg-dev" \
  --template-file infra/main.bicep \
  --parameters environmentName=dev
```

## Post-Deployment Configuration

After infrastructure deployment, the following is automatically configured:

### Automated Configuration

**RBAC for Managed Identity (Automated)**: The Bicep deployment automatically grants the App Service Managed Identity the "Azure Relay Owner" role on the Relay namespace. This allows the gateway application to:
- Create and manage Hybrid Connections
- Generate SAS tokens for listeners
- Send messages through Relay

No manual role assignment is needed.

### Manual Configuration Steps

The following steps still need to be performed manually:

### 1. Deploy Application Code

Deploy the gateway application to App Service using:
- Azure CLI: `az webapp deployment source config-zip`
- GitHub Actions (recommended for CI/CD)
- Azure DevOps Pipelines

### 2. Configure Application Settings

Set environment variables for the gateway application:

```bash
az webapp config appsettings set \
  --resource-group "azhexgate-rg-dev" \
  --name "azhexgate-app-dev" \
  --settings \
    RELAY_NAMESPACE="azhexgate-relay-dev" \
    ENVIRONMENT="dev"
```

## Cleanup

To remove all deployed resources:

```bash
az group delete \
  --name "azhexgate-rg-dev" \
  --yes \
  --no-wait
```

## Cost Estimation

Approximate monthly costs (East US region, as of 2026):

| Component | SKU/Tier | Estimated Cost |
|-----------|----------|----------------|
| Azure Relay | Standard | ~$10/month (base) + metered usage |
| App Service Plan | B1 (Basic) | ~$13/month |
| **Total** | | **~$23/month** + usage |

**Notes:**
- Relay charges additional fees based on message count and data transfer
- Actual costs depend on traffic volume and region
- Consider scaling to higher SKUs for production workloads

## Troubleshooting

### Deployment Fails with "Location not available"

Ensure the chosen location supports all required resource types:

```bash
az provider show \
  --namespace Microsoft.Relay \
  --query "resourceTypes[?resourceType=='namespaces'].locations"
```

### Permission Denied Errors

Verify you have Contributor role on the resource group:

```bash
az role assignment list \
  --resource-group "azhexgate-rg-dev" \
  --assignee "$(az account show --query user.name -o tsv)"
```

### Bicep Build Errors

Ensure Bicep CLI is up to date:

```bash
az bicep upgrade
az bicep version
```

## Next Steps

After infrastructure deployment:

1. Review `docs/architecture.md` for system architecture details
2. Deploy the gateway application code
3. Configure DNS and custom domains (future enhancement)
4. Set up TLS certificates (future enhancement)
5. Configure Azure Front Door for global distribution (future enhancement)

## Additional Resources

- [Azure Relay Documentation](https://docs.microsoft.com/azure/azure-relay/)
- [App Service Documentation](https://docs.microsoft.com/azure/app-service/)
- [Bicep Documentation](https://docs.microsoft.com/azure/azure-resource-manager/bicep/)
- [AzHexGate Architecture](./architecture.md)
