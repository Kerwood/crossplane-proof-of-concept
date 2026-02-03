package main

import (
	"fmt"

	"composer"
	"github.com/crossplane/function-sdk-go/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Service composes a Kubernetes Service from the XDeployment spec.
// The Service is only created when a port is specified.
type Service struct {
	XComposer
	ObservedResource *corev1.Service
}

// NewService creates a new Service composer. It looks up the observed Service
// resource and deserializes it for readiness checks. Returns an error if the
// observed resource exists but cannot be deserialized.
func NewService(f XContext) (*Service, error) {
	resourceName := resource.Name(fmt.Sprintf("xdeployment-service-%s", f.XR.Name))
	observedStructured, err := composer.ConvertObserved[corev1.Service](f.Observed, resourceName)
	if err != nil {
		return nil, err
	}

	return &Service{
		XComposer: XComposer{
			FunctionContext: f,
			ResourceName:    resourceName,
			ConditionType:   "ServiceReady",
		},
		ObservedResource: observedStructured,
	}, nil
}

// ComposeDesiredResource builds the desired Service and wraps it as a
// DesiredResource for inclusion in the function response. Returns nil if
// no port is specified in the XDeployment spec.
func (s *Service) ComposeDesiredResource() (*composer.DesiredResource, error) {
	return s.ComposeDesiredResourceFrom(s.createResource())
}

// isReady returns true if the observed Service has been assigned a ClusterIP,
// indicating that it is ready to receive traffic.
func (s *Service) IsReady() bool {
	if s.ObservedResource == nil {
		return false
	}
	return s.ObservedResource.Spec.ClusterIP != ""
}

// createResource constructs the Kubernetes Service spec from the XDeployment.
// Returns nil if no port is specified, signaling that the Service should not be created.
func (s *Service) createResource() *corev1.Service {
	log := s.FunctionContext.Log
	xd := s.FunctionContext.XR

	if xd.Spec.Port == nil {
		log.Debug("no port found, skipping service", "name", s.XComposer.ResourceName)
		return nil
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: xd.GetName(),
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app.kubernetes.io/name": xd.GetName(),
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       8080,
					TargetPort: intstr.FromInt(int(*xd.Spec.Port)),
				},
			},
		},
	}
}
