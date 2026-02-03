package main

import (
	"context"
	"testing"

	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	defaults "github.com/crossplane/function-xdeployment/input/v1beta1"
	"github.com/kerwood/crossplane-xrd-generator/resources/xdeployment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

func TestRunFunction(t *testing.T) {
	tests := []struct {
		name              string
		xDeployment       *xdeployment.XDeployment
		input             *defaults.XDeploymentDefaults
		wantResourceCount int
		wantConditions    []string
	}{
		{
			name: "composes deployment, service, and httproute",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image:    "nginx:latest",
					Port:     ptr.To(8080),
					Hostname: "test.example.com",
				},
			},
			input: &defaults.XDeploymentDefaults{
				Gateway: &defaults.GatewayConfig{
					Name:      "my-gateway",
					Namespace: "gateway-ns",
				},
			},
			wantResourceCount: 3,
			wantConditions:    []string{"DeploymentReady", "ServiceReady", "HttpRouteReady"},
		},
		{
			name: "composes only deployment when no port specified",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image: "nginx:latest",
				},
			},
			input:             &defaults.XDeploymentDefaults{},
			wantResourceCount: 1,
			wantConditions:    []string{"DeploymentReady"},
		},
		{
			name: "composes deployment and service when port specified but no hostname",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image: "nginx:latest",
					Port:  ptr.To(8080),
				},
			},
			input: &defaults.XDeploymentDefaults{
				Gateway: &defaults.GatewayConfig{
					Name:      "my-gateway",
					Namespace: "gateway-ns",
				},
			},
			wantResourceCount: 2,
			wantConditions:    []string{"DeploymentReady", "ServiceReady"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := buildRunFunctionRequest(t, tt.xDeployment, tt.input)

			f := &Function{log: logging.NewNopLogger()}
			rsp, err := f.RunFunction(context.Background(), req)

			require.NoError(t, err)
			require.NotNil(t, rsp)

			// Check that the expected number of resources were composed
			assert.Len(t, rsp.GetDesired().GetResources(), tt.wantResourceCount)

			// Check that conditions were set
			for _, condType := range tt.wantConditions {
				found := false
				for _, cond := range rsp.GetConditions() {
					if cond.Type == condType {
						found = true
						break
					}
				}
				assert.True(t, found, "expected condition %s not found", condType)
			}
		})
	}
}

func TestInternalErrorResponse(t *testing.T) {
	rsp := &fnv1.RunFunctionResponse{}
	testErr := assert.AnError

	InternalErrorResponse(rsp, testErr)

	// Check that a condition was set
	require.Len(t, rsp.GetConditions(), 1)
	cond := rsp.GetConditions()[0]
	assert.Equal(t, "FunctionSuccess", cond.Type)
	assert.Equal(t, fnv1.Status_STATUS_CONDITION_FALSE, cond.Status)
	assert.Equal(t, "InternalError", cond.Reason)

	// Check that fatal result was set
	require.Len(t, rsp.GetResults(), 1)
	assert.Equal(t, fnv1.Severity_SEVERITY_FATAL, rsp.GetResults()[0].Severity)
}

// buildRunFunctionRequest creates a RunFunctionRequest for testing
func buildRunFunctionRequest(t *testing.T, xd *xdeployment.XDeployment, input *defaults.XDeploymentDefaults) *fnv1.RunFunctionRequest {
	t.Helper()

	// Convert XDeployment to unstructured
	xdUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(xd)
	require.NoError(t, err)

	xdStruct, err := structpb.NewStruct(xdUnstructured)
	require.NoError(t, err)

	// Convert input to unstructured
	var inputStruct *structpb.Struct
	if input != nil {
		inputUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(input)
		require.NoError(t, err)
		inputStruct, err = structpb.NewStruct(inputUnstructured)
		require.NoError(t, err)
	}

	return &fnv1.RunFunctionRequest{
		Meta: &fnv1.RequestMeta{
			Tag: "test",
		},
		Input: inputStruct,
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: xdStruct,
			},
			Resources: map[string]*fnv1.Resource{},
		},
		Desired: &fnv1.State{
			Composite: &fnv1.Resource{
				Resource: xdStruct,
			},
			Resources: map[string]*fnv1.Resource{},
		},
	}
}
