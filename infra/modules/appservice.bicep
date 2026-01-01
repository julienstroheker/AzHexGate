// App Service Module
// Minimal App Service with Managed Identity for AzHexGate gateway

@description('Location for the App Service')
param location string = resourceGroup().location

@description('Name of the App Service plan')
param appServicePlanName string

@description('Name of the App Service')
param appServiceName string

@description('SKU for the App Service plan')
param skuName string = 'B1'

@description('SKU tier for the App Service plan')
param skuTier string = 'Basic'

@description('Tags to apply to resources')
param tags object = {}

@description('Runtime stack for the App Service')
param linuxFxVersion string = 'GO|1.23'

// App Service Plan
resource appServicePlan 'Microsoft.Web/serverfarms@2023-12-01' = {
  name: appServicePlanName
  location: location
  tags: tags
  sku: {
    name: skuName
    tier: skuTier
  }
  kind: 'linux'
  properties: {
    reserved: true
  }
}

// App Service with System-Assigned Managed Identity
resource appService 'Microsoft.Web/sites@2023-12-01' = {
  name: appServiceName
  location: location
  tags: tags
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    serverFarmId: appServicePlan.id
    httpsOnly: true
    siteConfig: {
      linuxFxVersion: linuxFxVersion
      alwaysOn: true
      http20Enabled: true
      minTlsVersion: '1.2'
      ftpsState: 'Disabled'
    }
  }
}

// Outputs
@description('Resource ID of the App Service Plan')
output appServicePlanId string = appServicePlan.id

@description('Name of the App Service Plan')
output appServicePlanName string = appServicePlan.name

@description('Resource ID of the App Service')
output appServiceId string = appService.id

@description('Name of the App Service')
output appServiceName string = appService.name

@description('Default hostname of the App Service')
output appServiceHostName string = appService.properties.defaultHostName

@description('Principal ID of the App Service Managed Identity')
output appServicePrincipalId string = appService.identity.principalId

@description('Tenant ID of the App Service Managed Identity')
output appServiceTenantId string = appService.identity.tenantId
