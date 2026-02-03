package main

import (
	"testing"

	"github.com/crossplane/function-sdk-go/logging"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/kerwood/crossplane-xrd-generator/resources/xdeployment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestNewDeployment(t *testing.T) {
	tests := []struct {
		name         string
		xDeployment  *xdeployment.XDeployment
		wantErr      bool
		wantResName  string
		wantCondType string
	}{
		{
			name: "creates deployment composer successfully",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
			},
			wantErr:      false,
			wantResName:  "xdeployment-deployment-test-app",
			wantCondType: "DeploymentReady",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := XContext{
				Observed: map[resource.Name]resource.ObservedComposed{},
				XR:       tt.xDeployment,
				Log:      logging.NewNopLogger(),
			}

			got, err := NewDeployment(ctx)

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

func TestDeployment_ComposeDesiredResource(t *testing.T) {
	tests := []struct {
		name         string
		xDeployment  *xdeployment.XDeployment
		wantNil      bool
		wantImage    string
		wantReplicas int32
	}{
		{
			name: "creates deployment with specified values",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image:    "nginx:latest",
					Replicas: ptr.To(int32(3)),
					Port:     ptr.To(8080),
				},
			},
			wantNil:      false,
			wantImage:    "nginx:latest",
			wantReplicas: 3,
		},
		{
			name: "defaults replicas to 2 when not specified",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image: "nginx:latest",
				},
			},
			wantNil:      false,
			wantImage:    "nginx:latest",
			wantReplicas: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := XContext{
				Observed: map[resource.Name]resource.ObservedComposed{},
				XR:       tt.xDeployment,
				Log:      logging.NewNopLogger(),
			}

			deployment, err := NewDeployment(ctx)
			require.NoError(t, err)

			got, err := deployment.ComposeDesiredResource()
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

func TestDeployment_IsReady(t *testing.T) {
	tests := []struct {
		name             string
		observedResource *appsv1.Deployment
		want             bool
	}{
		{
			name:             "returns false when observed resource is nil",
			observedResource: nil,
			want:             false,
		},
		{
			name: "returns false when no conditions",
			observedResource: &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					Conditions: []appsv1.DeploymentCondition{},
				},
			},
			want: false,
		},
		{
			name: "returns false when Available condition is False",
			observedResource: &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					Conditions: []appsv1.DeploymentCondition{
						{
							Type:   appsv1.DeploymentAvailable,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "returns true when Available condition is True",
			observedResource: &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					Conditions: []appsv1.DeploymentCondition{
						{
							Type:   appsv1.DeploymentAvailable,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Deployment{
				ObservedResource: tt.observedResource,
			}

			got := d.IsReady()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeployment_createResource(t *testing.T) {
	tests := []struct {
		name         string
		xDeployment  *xdeployment.XDeployment
		wantName     string
		wantReplicas int32
		wantPort     bool
		wantEnvCount int
	}{
		{
			name: "creates deployment with all fields",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image:    "myimage:v1",
					Replicas: ptr.To(int32(5)),
					Port:     ptr.To(3000),
					Env: map[string]string{
						"FOO": "bar",
						"BAZ": "qux",
					},
				},
			},
			wantName:     "my-app",
			wantReplicas: 5,
			wantPort:     true,
			wantEnvCount: 2,
		},
		{
			name: "creates deployment without port",
			xDeployment: &xdeployment.XDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "no-port-app",
				},
				Spec: xdeployment.XDeploymentSpec{
					Image: "myimage:v1",
				},
			},
			wantName:     "no-port-app",
			wantReplicas: 2, // default
			wantPort:     false,
			wantEnvCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := XContext{
				Observed: map[resource.Name]resource.ObservedComposed{},
				XR:       tt.xDeployment,
				Log:      logging.NewNopLogger(),
			}

			d := &Deployment{
				XComposer: XComposer{
					FunctionContext: ctx,
				},
			}

			got := d.createResource()

			require.NotNil(t, got)
			assert.Equal(t, tt.wantName, got.Name)
			assert.Equal(t, tt.wantReplicas, *got.Spec.Replicas)
			assert.Len(t, got.Spec.Template.Spec.Containers, 1)

			container := got.Spec.Template.Spec.Containers[0]
			if tt.wantPort {
				assert.Len(t, container.Ports, 1)
			} else {
				assert.Len(t, container.Ports, 0)
			}
			assert.Len(t, container.Env, tt.wantEnvCount)
		})
	}
}

func TestConvertEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		envMap   map[string]string
		wantLen  int
		wantKeys []string // expected order (sorted)
	}{
		{
			name:     "returns nil for empty map",
			envMap:   map[string]string{},
			wantLen:  0,
			wantKeys: nil,
		},
		{
			name:     "returns nil for nil map",
			envMap:   nil,
			wantLen:  0,
			wantKeys: nil,
		},
		{
			name: "sorts keys alphabetically",
			envMap: map[string]string{
				"ZEBRA": "z",
				"ALPHA": "a",
				"BETA":  "b",
			},
			wantLen:  3,
			wantKeys: []string{"ALPHA", "BETA", "ZEBRA"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertEnvVars(tt.envMap)

			if tt.wantLen == 0 {
				assert.Nil(t, got)
				return
			}

			require.Len(t, got, tt.wantLen)
			for i, key := range tt.wantKeys {
				assert.Equal(t, key, got[i].Name)
				assert.Equal(t, tt.envMap[key], got[i].Value)
			}
		})
	}
}
