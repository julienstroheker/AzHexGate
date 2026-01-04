// Main Bicep Template for AzHexGate Infrastructure MVP
// Deploys minimal Azure Relay and App Service with Managed Identity

targetScope = 'resourceGroup'

@description('Environment name (e.g., dev, prod)')
@minLength(2)
@maxLength(10)
param environmentName string = 'dev'

@description('Location for all resources')
param location string = resourceGroup().location

@description('Base name for resources')
param baseName string = 'azhexgate'

@description('Tags to apply to all resources')
param tags object = {
  environment: environmentName
  project: 'AzHexGate'
  managedBy: 'Bicep'
}

// Resource naming
var relayNamespaceName = '${baseName}-relay-${environmentName}'
var appServicePlanName = '${baseName}-plan-${environmentName}'
var appServiceName = '${baseName}-app-${environmentName}'

// Deploy Azure Relay Namespace
module relay 'modules/relay.bicep' = {
  name: 'relay-deployment'
  params: {
    location: location
    relayNamespaceName: relayNamespaceName
    skuName: 'Standard'
    tags: tags
  }
}

// Deploy App Service with Managed Identity
module appService 'modules/appservice.bicep' = {
  name: 'appservice-deployment'
  params: {
    location: location
    appServicePlanName: appServicePlanName
    appServiceName: appServiceName
    skuName: 'B1'
    skuTier: 'Basic'
    linuxFxVersion: 'GO|1.23'
    tags: tags
  }
}

// Grant App Service Managed Identity access to Relay
module relayRbac 'modules/relay-rbac.bicep' = {
  name: 'relay-rbac-deployment'
  params: {
    relayNamespaceName: relay.outputs.relayNamespaceName
    appServicePrincipalId: appService.outputs.appServicePrincipalId
  }
}

// Outputs
@description('Relay namespace name')
output relayNamespaceName string = relay.outputs.relayNamespaceName

@description('Relay namespace endpoint')
output relayNamespaceEndpoint string = relay.outputs.relayNamespaceEndpoint

@description('App Service name')
output appServiceName string = appService.outputs.appServiceName

@description('App Service hostname')
output appServiceHostName string = appService.outputs.appServiceHostName

@description('App Service Managed Identity Principal ID')
output appServicePrincipalId string = appService.outputs.appServicePrincipalId
