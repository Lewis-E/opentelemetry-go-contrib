// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azurefunctions

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func TestNewResourceDetector(t *testing.T) {
	assert.IsType(t, &ResourceDetector{}, NewResourceDetector())
}

func TestReturnsEmptyIfNoFunctionsMarker(t *testing.T) {
	t.Setenv(functionsWorkerRuntimeEnvVar, "")
	t.Setenv(functionsExtensionVersionEnvVar, "")
	t.Setenv(websiteSiteNameEnvVar, "my-function-app")

	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectGatesOnExtensionVersion(t *testing.T) {
	t.Setenv(functionsWorkerRuntimeEnvVar, "")
	t.Setenv(functionsExtensionVersionEnvVar, "~4")
	t.Setenv(computerNameEnvVar, "")
	t.Setenv(websiteSiteNameEnvVar, "my-function-app")

	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureFunctions,
		semconv.ServiceName("my-function-app"),
	)
	assert.Equal(t, expected, res)
}

func TestDetectFullAttributeSet(t *testing.T) {
	t.Setenv(functionsWorkerRuntimeEnvVar, "dotnet-isolated")
	t.Setenv(regionNameEnvVar, "eastus")
	t.Setenv(websiteSiteNameEnvVar, "my-function-app")
	t.Setenv(websiteResourceGroupEnvVar, "my-rg")
	t.Setenv(websiteOwnerNameEnvVar, "11111111-1111-1111-1111-111111111111+my-rg-EastUSwebspace")
	t.Setenv(websiteInstanceIDEnvVar, "a1b2c3")
	t.Setenv(computerNameEnvVar, "ln1sdlwk0006CQ")
	t.Setenv(websiteSlotNameEnvVar, "staging")

	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureFunctions,
		semconv.CloudRegion("eastus"),
		semconv.ServiceName("my-function-app"),
		semconv.AzureResourceGroupName("my-rg"),
		semconv.CloudAccountID("11111111-1111-1111-1111-111111111111"),
		semconv.CloudResourceID("/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/my-rg/providers/Microsoft.Web/sites/my-function-app"),
		semconv.FaaSInstance("a1b2c3"),
		instanceNameKey.String("ln1sdlwk0006CQ"),
		semconv.DeploymentEnvironmentNameKey.String("staging"),
	)
	assert.Equal(t, expected, res)
}

func TestDetectOwnerNameWithoutPlus(t *testing.T) {
	t.Setenv(functionsWorkerRuntimeEnvVar, "dotnet-isolated")
	t.Setenv(websiteSiteNameEnvVar, "my-function-app")
	t.Setenv(websiteResourceGroupEnvVar, "my-rg")
	t.Setenv(websiteOwnerNameEnvVar, "11111111-1111-1111-1111-111111111111")
	t.Setenv(computerNameEnvVar, "")

	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureFunctions,
		semconv.ServiceName("my-function-app"),
		semconv.AzureResourceGroupName("my-rg"),
		semconv.CloudAccountID("11111111-1111-1111-1111-111111111111"),
		semconv.CloudResourceID("/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/my-rg/providers/Microsoft.Web/sites/my-function-app"),
	)
	assert.Equal(t, expected, res)
}

func TestDetectFlexConsumptionMissingResourceGroup(t *testing.T) {
	// Flex Consumption does not expose WEBSITE_RESOURCE_GROUP, so
	// cloud.resource_id and azure.resource_group.name must be omitted
	// rather than emitted with a partial value; cloud.account.id is still
	// derived from WEBSITE_OWNER_NAME independently.
	t.Setenv(functionsWorkerRuntimeEnvVar, "dotnet-isolated")
	t.Setenv(websiteSiteNameEnvVar, "my-function-app")
	t.Setenv(websiteOwnerNameEnvVar, "11111111-1111-1111-1111-111111111111+my-rg-EastUSwebspace")
	t.Setenv(computerNameEnvVar, "")
	t.Setenv(websitePodNameEnvVar, "pod-abc123")
	t.Setenv(containerNameEnvVar, "should-not-be-used")

	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureFunctions,
		semconv.ServiceName("my-function-app"),
		semconv.CloudAccountID("11111111-1111-1111-1111-111111111111"),
		semconv.FaaSInstance("pod-abc123"),
		instanceNameKey.String("pod-abc123"),
	)
	assert.Equal(t, expected, res)
}

func TestFunctionsInstanceIDFallsBackToContainerName(t *testing.T) {
	t.Setenv(functionsWorkerRuntimeEnvVar, "dotnet-isolated")
	t.Setenv(websiteSiteNameEnvVar, "my-function-app")
	t.Setenv(computerNameEnvVar, "")
	t.Setenv(containerNameEnvVar, "container-xyz")

	detector := ResourceDetector{}
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureFunctions,
		semconv.ServiceName("my-function-app"),
		semconv.FaaSInstance("container-xyz"),
	)
	assert.Equal(t, expected, res)
}

func TestFunctionsInstanceName(t *testing.T) {
	tests := []struct {
		name          string
		computerName  string
		podName       string
		containerName string
		want          string
	}{
		{
			name:          "COMPUTERNAME wins",
			computerName:  "pl0mdlwk0002M1",
			podName:       "pod-abc123",
			containerName: "container-xyz",
			want:          "pl0mdlwk0002M1",
		},
		{
			name:          "falls back to WEBSITE_POD_NAME",
			podName:       "pod-abc123",
			containerName: "container-xyz",
			want:          "pod-abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(functionsWorkerRuntimeEnvVar, "dotnet-isolated")
			t.Setenv(computerNameEnvVar, tt.computerName)
			t.Setenv(websitePodNameEnvVar, tt.podName)
			t.Setenv(containerNameEnvVar, tt.containerName)

			res, err := NewResourceDetector().Detect(t.Context())

			assert.NoError(t, err)
			got, ok := res.Set().Value(instanceNameKey)
			assert.True(t, ok)
			assert.Equal(t, tt.want, got.AsString())
		})
	}
}

func TestFunctionsInstanceNameIgnoresContainerName(t *testing.T) {
	t.Setenv(functionsWorkerRuntimeEnvVar, "dotnet-isolated")
	t.Setenv(computerNameEnvVar, "")
	t.Setenv(websitePodNameEnvVar, "")
	t.Setenv(containerNameEnvVar, "container-xyz")

	res, err := NewResourceDetector().Detect(t.Context())

	assert.NoError(t, err)
	assert.False(t, res.Set().HasValue(instanceNameKey))
}

func TestFaaSInstanceIgnoresComputerName(t *testing.T) {
	t.Setenv(functionsWorkerRuntimeEnvVar, "dotnet-isolated")
	t.Setenv(websiteInstanceIDEnvVar, "a1b2c3")
	t.Setenv(computerNameEnvVar, "pl0mdlwk0002M1")

	res, err := NewResourceDetector().Detect(t.Context())

	assert.NoError(t, err)
	faasInstance, ok := res.Set().Value(semconv.FaaSInstanceKey)
	assert.True(t, ok)
	assert.Equal(t, "a1b2c3", faasInstance.AsString())
	instanceName, ok := res.Set().Value(instanceNameKey)
	assert.True(t, ok)
	assert.Equal(t, "pl0mdlwk0002M1", instanceName.AsString())
}

func TestWithAttributeFilter(t *testing.T) {
	t.Setenv(functionsWorkerRuntimeEnvVar, "dotnet-isolated")
	t.Setenv(websiteSiteNameEnvVar, "my-function-app")
	t.Setenv(regionNameEnvVar, "eastus")

	filter := func(kv attribute.KeyValue) bool {
		return strings.HasPrefix(string(kv.Key), "cloud.")
	}
	detector := NewResourceDetector(WithAttributeFilter(filter))
	res, err := detector.Detect(t.Context())

	assert.NoError(t, err)
	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureFunctions,
		semconv.CloudRegion("eastus"),
	)
	assert.Equal(t, expected, res)
}
