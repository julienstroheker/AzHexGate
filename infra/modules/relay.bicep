// Relay Namespace Module
// Minimal Azure Relay namespace for AzHexGate tunnel infrastructure

@description('Location for the Relay namespace')
param location string = resourceGroup().location

@description('Name of the Relay namespace')
param relayNamespaceName string

@description('SKU for the Relay namespace')
@allowed([
  'Standard'
])
param skuName string = 'Standard'

@description('Tags to apply to the Relay namespace')
param tags object = {}

// Azure Relay Namespace
resource relayNamespace 'Microsoft.Relay/namespaces@2021-11-01' = {
  name: relayNamespaceName
  location: location
  tags: tags
  sku: {
    name: skuName
    tier: skuName
  }
  properties: {}
}

// Outputs
@description('Resource ID of the Relay namespace')
output relayNamespaceId string = relayNamespace.id

@description('Name of the Relay namespace')
output relayNamespaceName string = relayNamespace.name

@description('Endpoint of the Relay namespace')
output relayNamespaceEndpoint string = relayNamespace.properties.serviceBusEndpoint
