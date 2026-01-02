// Relay RBAC Module
// Assigns Azure Relay Owner role to App Service Managed Identity

@description('Name of the Relay namespace')
param relayNamespaceName string

@description('Principal ID of the App Service Managed Identity')
param appServicePrincipalId string

// Reference existing Relay namespace
resource relayNamespace 'Microsoft.Relay/namespaces@2021-11-01' existing = {
  name: relayNamespaceName
}

// Azure Relay Owner role definition
// This built-in role allows full management of Relay resources
var relayOwnerRoleId = '2787bf04-f1f5-4bfe-8383-c8a24483ee38'

// Assign Azure Relay Owner role to App Service Managed Identity
resource relayRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(relayNamespace.id, appServicePrincipalId, relayOwnerRoleId)
  scope: relayNamespace
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', relayOwnerRoleId)
    principalId: appServicePrincipalId
    principalType: 'ServicePrincipal'
  }
}

// Outputs
@description('Role assignment ID')
output roleAssignmentId string = relayRoleAssignment.id

@description('Role assignment name')
output roleAssignmentName string = relayRoleAssignment.name
