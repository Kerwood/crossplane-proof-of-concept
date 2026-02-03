package main

import (
	"testing"

	"github.com/crossplane/function-sdk-go/logging"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/kerwood/crossplane-xrd-generator/resources/xdeployment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name         string
		xDeployment  *xdeployment.XDeployment
		wantErr      bool
		wantResName  string
		wantCondType string
	}{
		{
			name: "creates service composer successfully",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
			},
			wantErr:      false,
			wantResName:  "xdeployment-service-test-app",
			wantCondType: "ServiceReady",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := XContext{
				Observed: map[resource.Name]resource.ObservedComposed{},
				XR:       tt.xDeployment,
				Log:      logging.NewNopLogger(),
			}

			got, err := NewService(ctx)

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

func TestService_ComposeDesiredResource(t *testing.T) {
	tests := []struct {
		name        string
		xDeployment *xdeployment.XDeployment
		wantNil     bool
	}{
		{
			name: "creates service when port is specified",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image: "nginx:latest",
					Port:  ptr.To(8080),
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
					Image: "nginx:latest",
				},
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := XContext{
				Observed: map[resource.Name]resource.ObservedComposed{},
				XR:       tt.xDeployment,
				Log:      logging.NewNopLogger(),
			}

			service, err := NewService(ctx)
			require.NoError(t, err)

			got, err := service.ComposeDesiredResource()
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

func TestService_IsReady(t *testing.T) {
	tests := []struct {
		name             string
		observedResource *corev1.Service
		want             bool
	}{
		{
			name:             "returns false when observed resource is nil",
			observedResource: nil,
			want:             false,
		},
		{
			name: "returns false when ClusterIP is empty",
			observedResource: &corev1.Service{
				Spec: corev1.ServiceSpec{
					ClusterIP: "",
				},
			},
			want: false,
		},
		{
			name: "returns true when ClusterIP is assigned",
			observedResource: &corev1.Service{
				Spec: corev1.ServiceSpec{
					ClusterIP: "10.0.0.1",
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				ObservedResource: tt.observedResource,
			}

			got := s.IsReady()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestService_createResource(t *testing.T) {
	tests := []struct {
		name        string
		xDeployment *xdeployment.XDeployment
		wantNil     bool
		wantName    string
		wantPort    int32
		wantTarget  int
	}{
		{
			name: "creates service with specified port",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image: "myimage:v1",
					Port:  ptr.To(3000),
				},
			},
			wantNil:    false,
			wantName:   "my-app",
			wantPort:   8080,
			wantTarget: 3000,
		},
		{
			name: "returns nil when port not specified",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image: "myimage:v1",
				},
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := XContext{
				Observed: map[resource.Name]resource.ObservedComposed{},
				XR:       tt.xDeployment,
				Log:      logging.NewNopLogger(),
			}

			s := &Service{
				XComposer: XComposer{
					FunctionContext: ctx,
				},
			}

			got := s.createResource()

			if tt.wantNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, tt.wantName, got.Name)
			assert.Equal(t, corev1.ServiceTypeClusterIP, got.Spec.Type)
			require.Len(t, got.Spec.Ports, 1)
			assert.Equal(t, tt.wantPort, got.Spec.Ports[0].Port)
			assert.Equal(t, tt.wantTarget, got.Spec.Ports[0].TargetPort.IntValue())
			assert.Equal(t, tt.wantName, got.Spec.Selector["app.kubernetes.io/name"])
		})
	}
}
