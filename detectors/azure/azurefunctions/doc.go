// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

/*
Package azurefunctions provides a resource detector which supports
detecting attributes specific to Azure Functions.

The detector gates on FUNCTIONS_WORKER_RUNTIME or
FUNCTIONS_EXTENSION_VERSION, since Azure Functions shares most WEBSITE_*
environment variables with Azure App Service.

According to semantic conventions for [cloud], [service], and [faas]
attributes, and custom attributes for the Azure resource group and
instance name, the following attributes are added when running on Azure
Functions:

  - cloud.provider
  - cloud.platform
  - cloud.region
  - cloud.account.id
  - cloud.resource_id
  - service.name
  - azure.resource_group.name
  - faas.instance
  - azure.app_service.instance.name
  - deployment.environment.name

faas.instance comes from the first non-empty value of WEBSITE_INSTANCE_ID,
WEBSITE_POD_NAME, and CONTAINER_NAME. azure.app_service.instance.name comes
from the first non-empty value of COMPUTERNAME and WEBSITE_POD_NAME, emitted
as Azure sets it, which is the name Azure Monitor reports in its Instance
dimension.

cloud.resource_id, azure.resource_group.name, faas.instance, and
azure.app_service.instance.name are sourced defensively and vary by hosting
plan, so they may be absent from the resulting resource.

[cloud]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/cloud.md
[service]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/service.md
[faas]: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/resource/faas.md
*/
package azurefunctions
