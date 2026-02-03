package composer

import (
	"testing"

	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	// Register corev1 types with the scheme for testing
	_ = corev1.AddToScheme(composed.Scheme)
}

// TestFunctionContext holds test types for generic FunctionContext
type TestXR struct {
	Name string
}

type TestDefaults struct {
	Value string
}

func TestBaseComposer_GetConditionType(t *testing.T) {
	tests := []struct {
		name          string
		conditionType string
		want          string
	}{
		{
			name:          "returns configured condition type",
			conditionType: "DeploymentReady",
			want:          "DeploymentReady",
		},
		{
			name:          "returns empty string when not set",
			conditionType: "",
			want:          "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BaseComposer[TestXR, TestDefaults]{
				ConditionType: tt.conditionType,
			}

			got := b.GetConditionType()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBaseComposer_ComposeDesiredResourceFrom(t *testing.T) {
	tests := []struct {
		name               string
		structuredResource runtime.Object
		resourceName       resource.Name
		wantNil            bool
		wantErr            bool
	}{
		{
			name:               "returns nil for nil resource",
			structuredResource: nil,
			resourceName:       "test-resource",
			wantNil:            true,
			wantErr:            false,
		},
		{
			name:               "returns nil for typed nil pointer",
			structuredResource: (*corev1.Service)(nil),
			resourceName:       "test-resource",
			wantNil:            true,
			wantErr:            false,
		},
		{
			name: "composes valid service resource",
			structuredResource: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-service",
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
				},
			},
			resourceName: "test-resource",
			wantNil:      false,
			wantErr:      false,
		},
		{
			name: "composes valid configmap resource",
			structuredResource: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-configmap",
				},
				Data: map[string]string{
					"key": "value",
				},
			},
			resourceName: "configmap-resource",
			wantNil:      false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := FunctionContext[TestXR, TestDefaults]{
				FunctionResponse: &fnv1.RunFunctionResponse{},
				Log:              logging.NewNopLogger(),
			}

			b := &BaseComposer[TestXR, TestDefaults]{
				FunctionContext: ctx,
				ResourceName:    tt.resourceName,
				ConditionType:   "TestReady",
			}

			got, err := b.ComposeDesiredResourceFrom(tt.structuredResource)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, tt.resourceName, got.Name)
			assert.NotNil(t, got.Resource)
			assert.Equal(t, resource.ReadyFalse, got.Resource.Ready)
		})
	}
}

func TestConvertObserved(t *testing.T) {
	tests := []struct {
		name         string
		observed     map[resource.Name]resource.ObservedComposed
		resourceName resource.Name
		wantNil      bool
		wantErr      bool
		wantName     string
	}{
		{
			name:         "returns nil when resource not in observed map",
			observed:     map[resource.Name]resource.ObservedComposed{},
			resourceName: "missing-resource",
			wantNil:      true,
			wantErr:      false,
		},
		{
			name:         "returns nil when observed map is empty",
			observed:     map[resource.Name]resource.ObservedComposed{},
			resourceName: "any-resource",
			wantNil:      true,
			wantErr:      false,
		},
		{
			name: "converts observed service successfully",
			observed: map[resource.Name]resource.ObservedComposed{
				"test-service": {
					Resource: buildUnstructuredService("my-service", "10.0.0.1"),
				},
			},
			resourceName: "test-service",
			wantNil:      false,
			wantErr:      false,
			wantName:     "my-service",
		},
		{
			name: "converts observed configmap successfully",
			observed: map[resource.Name]resource.ObservedComposed{
				"test-configmap": {
					Resource: buildUnstructuredConfigMap("my-configmap"),
				},
			},
			resourceName: "test-configmap",
			wantNil:      false,
			wantErr:      false,
			wantName:     "my-configmap",
		},
		{
			name: "returns nil when looking for wrong resource name",
			observed: map[resource.Name]resource.ObservedComposed{
				"existing-resource": {
					Resource: buildUnstructuredService("my-service", "10.0.0.1"),
				},
			},
			resourceName: "different-resource",
			wantNil:      true,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertObserved[corev1.Service](tt.observed, tt.resourceName)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, tt.wantName, got.Name)
		})
	}
}

func TestConvertObserved_ConfigMap(t *testing.T) {
	observed := map[resource.Name]resource.ObservedComposed{
		"test-configmap": {
			Resource: buildUnstructuredConfigMap("my-configmap"),
		},
	}

	got, err := ConvertObserved[corev1.ConfigMap](observed, "test-configmap")

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "my-configmap", got.Name)
	assert.Equal(t, "test-value", got.Data["test-key"])
}

func TestConvertObserved_TypeMismatch(t *testing.T) {
	// Create a Service in the observed map
	observed := map[resource.Name]resource.ObservedComposed{
		"test-resource": {
			Resource: buildUnstructuredService("my-service", "10.0.0.1"),
		},
	}

	// Try to convert it as a ConfigMap - should succeed but with empty/default fields
	// because FromUnstructured is lenient
	got, err := ConvertObserved[corev1.ConfigMap](observed, "test-resource")

	// FromUnstructured doesn't error on type mismatch, it just sets fields it can
	require.NoError(t, err)
	require.NotNil(t, got)
	// The name field is common to all resources so it gets set
	assert.Equal(t, "my-service", got.Name)
}

func TestFunctionContext_Fields(t *testing.T) {
	xr := TestXR{Name: "test-xr"}
	defaults := TestDefaults{Value: "test-value"}
	observed := map[resource.Name]resource.ObservedComposed{}
	rsp := &fnv1.RunFunctionResponse{}
	log := logging.NewNopLogger()

	ctx := FunctionContext[TestXR, TestDefaults]{
		Observed:         observed,
		FunctionResponse: rsp,
		XR:               xr,
		Defaults:         defaults,
		Log:              log,
	}

	assert.Equal(t, xr, ctx.XR)
	assert.Equal(t, defaults, ctx.Defaults)
	assert.Equal(t, observed, ctx.Observed)
	assert.Equal(t, rsp, ctx.FunctionResponse)
	assert.NotNil(t, ctx.Log)
}

func TestDesiredResource_Fields(t *testing.T) {
	name := resource.Name("test-resource")
	desiredComposed := &resource.DesiredComposed{
		Ready: resource.ReadyTrue,
	}

	dr := DesiredResource{
		Name:     name,
		Resource: desiredComposed,
	}

	assert.Equal(t, name, dr.Name)
	assert.Equal(t, desiredComposed, dr.Resource)
	assert.Equal(t, resource.ReadyTrue, dr.Resource.Ready)
}

// Helper functions to build unstructured resources for testing

func buildUnstructuredService(name, clusterIP string) *composed.Unstructured {
	u := &composed.Unstructured{
		Unstructured: unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]any{
					"name": name,
				},
				"spec": map[string]any{
					"type":      "ClusterIP",
					"clusterIP": clusterIP,
				},
			},
		},
	}
	return u
}

func buildUnstructuredConfigMap(name string) *composed.Unstructured {
	u := &composed.Unstructured{
		Unstructured: unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]any{
					"name": name,
				},
				"data": map[string]any{
					"test-key": "test-value",
				},
			},
		},
	}
	return u
}
