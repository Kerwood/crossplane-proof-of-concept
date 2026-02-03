package composer

import (
	"reflect"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/response"
	"k8s.io/apimachinery/pkg/runtime"
)

// FunctionContext holds the shared state passed to all resource composers,
// including the observed resources, the function response for setting
// conditions, and the deserialized XDeployment spec.
type FunctionContext[XR any, D any] struct {
	Observed         map[resource.Name]resource.ObservedComposed
	FunctionResponse *fnv1.RunFunctionResponse
	XR               XR
	Defaults         D
	Log              logging.Logger
}

// DesiredResource pairs a resource name with its desired composed state.
// It is returned by each composer and used to populate the function response.
type DesiredResource struct {
	Name     resource.Name
	Resource *resource.DesiredComposed
}

// ComposableResource is the interface that all resource composers must implement.
// It provides methods to compose the desired resource, check its observed
// readiness, and return its condition type for status reporting.
type ComposableResource interface {
	// ComposeDesiredResource builds and returns the desired resource.
	// Returns nil if the resource should not be created (e.g. optional field not set).
	ComposeDesiredResource() (*DesiredResource, error)

	// IsReady checks the observed resource state and returns true if the
	// resource is available and healthy.
	IsReady() bool

	// GetConditionType returns the condition type string used when setting
	// status conditions on the composite resource (e.g. "DeploymentReady").
	GetConditionType() string
}

// BaseComposer provides shared fields and methods for all resource composers.
// Embed this in concrete composer structs to inherit common functionality.
type BaseComposer[XR any, D any] struct {
	FunctionContext FunctionContext[XR, D]
	ResourceName    resource.Name
	ConditionType   string
}

// GetConditionType returns the condition type used for status reporting
// on the composite resource.
func (b *BaseComposer[XR, D]) GetConditionType() string {
	return b.ConditionType
}

// ComposeDesiredResourceFrom converts a structured Kubernetes resource into a
// DesiredResource. It handles typed nil pointers (e.g. a *corev1.Service that
// is nil) by returning nil, signaling that the resource should be skipped.
func (b *BaseComposer[XR, D]) ComposeDesiredResourceFrom(structuredResource runtime.Object) (*DesiredResource, error) {
	if structuredResource == nil || reflect.ValueOf(structuredResource).IsNil() {
		return nil, nil
	}

	composed, err := composed.From(structuredResource)
	if err != nil {
		response.Fatal(b.FunctionContext.FunctionResponse,
			errors.Wrapf(err, "cannot convert %T to composed resource", structuredResource))
		return nil, err
	}

	return &DesiredResource{
		Name:     b.ResourceName,
		Resource: &resource.DesiredComposed{Resource: composed, Ready: resource.ReadyFalse},
	}, nil
}

// ConvertObserved looks up the observed resource by name and deserializes it
// into the specified type T. Returns nil if the resource has not yet been
// observed (e.g. on first reconciliation).
func ConvertObserved[T any](observed map[resource.Name]resource.ObservedComposed, resourceName resource.Name) (*T, error) {
	if obs, exists := observed[resourceName]; exists {
		var observedResource T

		err := runtime.DefaultUnstructuredConverter.FromUnstructured(
			obs.Resource.UnstructuredContent(),
			&observedResource,
		)
		if err != nil {
			return nil, err
		}

		return &observedResource, nil
	}

	return nil, nil
}
