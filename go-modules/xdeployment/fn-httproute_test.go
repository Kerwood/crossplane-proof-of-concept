package main

import (
	"testing"

	"github.com/crossplane/function-sdk-go/logging"
	"github.com/crossplane/function-sdk-go/resource"
	defaults "github.com/crossplane/function-xdeployment/input/v1beta1"
	"github.com/kerwood/crossplane-xrd-generator/resources/xdeployment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestNewHttpRoute(t *testing.T) {
	tests := []struct {
		name         string
		xDeployment  *xdeployment.XDeployment
		wantErr      bool
		wantResName  string
		wantCondType string
	}{
		{
			name: "creates httproute composer successfully",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
			},
			wantErr:      false,
			wantResName:  "xdeployment-httproute-test-app",
			wantCondType: "HttpRouteReady",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := XContext{
				Observed: map[resource.Name]resource.ObservedComposed{},
				XR:       tt.xDeployment,
				Log:      logging.NewNopLogger(),
			}

			got, err := NewHttpRoute(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, resource.Name(tt.wantResName), got.ResourceName)
			assert.Equal(t, tt.wantCondType, got.ConditionType)
		})
	}
}

func TestHttpRoute_ComposeDesiredResource(t *testing.T) {
	tests := []struct {
		name        string
		xDeployment *xdeployment.XDeployment
		defaults    *defaults.XDeploymentDefaults
		wantNil     bool
	}{
		{
			name: "creates httproute when port, hostname, and gateway are specified",
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
			defaults: &defaults.XDeploymentDefaults{
				Gateway: &defaults.GatewayConfig{
					Name:      "my-gateway",
					Namespace: "gateway-ns",
				},
			},
			wantNil: false,
		},
		{
			name: "returns nil when port is not specified",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image:    "nginx:latest",
					Hostname: "test.example.com",
				},
			},
			defaults: &defaults.XDeploymentDefaults{
				Gateway: &defaults.GatewayConfig{
					Name:      "my-gateway",
					Namespace: "gateway-ns",
				},
			},
			wantNil: true,
		},
		{
			name: "returns nil when hostname is not specified",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image: "nginx:latest",
					Port:  ptr.To(8080),
				},
			},
			defaults: &defaults.XDeploymentDefaults{
				Gateway: &defaults.GatewayConfig{
					Name:      "my-gateway",
					Namespace: "gateway-ns",
				},
			},
			wantNil: true,
		},
		{
			name: "returns nil when defaults is nil",
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
			defaults: nil,
			wantNil:  true,
		},
		{
			name: "returns nil when gateway config is nil",
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
			defaults: &defaults.XDeploymentDefaults{
				Gateway: nil,
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := XContext{
				Observed: map[resource.Name]resource.ObservedComposed{},
				XR:       tt.xDeployment,
				Defaults: tt.defaults,
				Log:      logging.NewNopLogger(),
			}

			httpRoute, err := NewHttpRoute(ctx)
			require.NoError(t, err)

			got, err := httpRoute.ComposeDesiredResource()
			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, resource.ReadyFalse, got.Resource.Ready)
		})
	}
}

func TestHttpRoute_IsReady(t *testing.T) {
	tests := []struct {
		name             string
		observedResource *gwapiv1.HTTPRoute
		want             bool
	}{
		{
			name:             "returns false when observed resource is nil",
			observedResource: nil,
			want:             false,
		},
		{
			name: "returns false when no parent status",
			observedResource: &gwapiv1.HTTPRoute{
				Status: gwapiv1.HTTPRouteStatus{
					RouteStatus: gwapiv1.RouteStatus{
						Parents: []gwapiv1.RouteParentStatus{},
					},
				},
			},
			want: false,
		},
		{
			name: "returns false when Accepted condition is False",
			observedResource: &gwapiv1.HTTPRoute{
				Status: gwapiv1.HTTPRouteStatus{
					RouteStatus: gwapiv1.RouteStatus{
						Parents: []gwapiv1.RouteParentStatus{
							{
								Conditions: []metav1.Condition{
									{
										Type:   string(gwapiv1.RouteConditionAccepted),
										Status: metav1.ConditionFalse,
									},
								},
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "returns true when Accepted condition is True",
			observedResource: &gwapiv1.HTTPRoute{
				Status: gwapiv1.HTTPRouteStatus{
					RouteStatus: gwapiv1.RouteStatus{
						Parents: []gwapiv1.RouteParentStatus{
							{
								Conditions: []metav1.Condition{
									{
										Type:   string(gwapiv1.RouteConditionAccepted),
										Status: metav1.ConditionTrue,
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "returns true when at least one parent has Accepted=True",
			observedResource: &gwapiv1.HTTPRoute{
				Status: gwapiv1.HTTPRouteStatus{
					RouteStatus: gwapiv1.RouteStatus{
						Parents: []gwapiv1.RouteParentStatus{
							{
								Conditions: []metav1.Condition{
									{
										Type:   string(gwapiv1.RouteConditionAccepted),
										Status: metav1.ConditionFalse,
									},
								},
							},
							{
								Conditions: []metav1.Condition{
									{
										Type:   string(gwapiv1.RouteConditionAccepted),
										Status: metav1.ConditionTrue,
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HttpRoute{
				ObservedResource: tt.observedResource,
			}

			got := h.IsReady()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHttpRoute_createResource(t *testing.T) {
	tests := []struct {
		name          string
		xDeployment   *xdeployment.XDeployment
		defaults      *defaults.XDeploymentDefaults
		wantNil       bool
		wantName      string
		wantHostname  string
		wantGateway   string
		wantGatewayNs string
	}{
		{
			name: "creates httproute with all required fields",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image:    "myimage:v1",
					Port:     ptr.To(3000),
					Hostname: "my-app.example.com",
				},
			},
			defaults: &defaults.XDeploymentDefaults{
				Gateway: &defaults.GatewayConfig{
					Name:      "prod-gateway",
					Namespace: "gateway-system",
				},
			},
			wantNil:       false,
			wantName:      "my-app",
			wantHostname:  "my-app.example.com",
			wantGateway:   "prod-gateway",
			wantGatewayNs: "gateway-system",
		},
		{
			name: "returns nil when port not specified",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image:    "myimage:v1",
					Hostname: "my-app.example.com",
				},
			},
			defaults: &defaults.XDeploymentDefaults{
				Gateway: &defaults.GatewayConfig{
					Name:      "prod-gateway",
					Namespace: "gateway-system",
				},
			},
			wantNil: true,
		},
		{
			name: "returns nil when hostname not specified",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image: "myimage:v1",
					Port:  ptr.To(3000),
				},
			},
			defaults: &defaults.XDeploymentDefaults{
				Gateway: &defaults.GatewayConfig{
					Name:      "prod-gateway",
					Namespace: "gateway-system",
				},
			},
			wantNil: true,
		},
		{
			name: "returns nil when defaults is nil",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image:    "myimage:v1",
					Port:     ptr.To(3000),
					Hostname: "my-app.example.com",
				},
			},
			defaults: nil,
			wantNil:  true,
		},
		{
			name: "returns nil when gateway is nil",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image:    "myimage:v1",
					Port:     ptr.To(3000),
					Hostname: "my-app.example.com",
				},
			},
			defaults: &defaults.XDeploymentDefaults{
				Gateway: nil,
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := XContext{
				Observed: map[resource.Name]resource.ObservedComposed{},
				XR:       tt.xDeployment,
				Defaults: tt.defaults,
				Log:      logging.NewNopLogger(),
			}

			h := &HttpRoute{
				XComposer: XComposer{
					FunctionContext: ctx,
				},
			}

			got := h.createResource()

			if tt.wantNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, tt.wantName, got.Name)
			require.Len(t, got.Spec.Hostnames, 1)
			assert.Equal(t, gwapiv1.Hostname(tt.wantHostname), got.Spec.Hostnames[0])
			require.Len(t, got.Spec.ParentRefs, 1)
			assert.Equal(t, gwapiv1.ObjectName(tt.wantGateway), got.Spec.ParentRefs[0].Name)
			assert.Equal(t, gwapiv1.Namespace(tt.wantGatewayNs), *got.Spec.ParentRefs[0].Namespace)
		})
	}
}
