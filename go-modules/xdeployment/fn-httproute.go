package main

import (
	"fmt"

	"composer"
	"github.com/crossplane/function-sdk-go/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// HttpRoute composes a Gateway API HTTPRoute from the XDeployment spec.
type HttpRoute struct {
	XComposer
	ObservedResource *gwapiv1.HTTPRoute
}

// NewHttpRoute creates a new HttpRoute composer. It looks up the observed
// HTTPRoute resource and deserializes it for readiness checks. Returns an
// error if the observed resource exists but cannot be deserialized.
func NewHttpRoute(f XContext) (*HttpRoute, error) {
	resourceName := resource.Name(fmt.Sprintf("xdeployment-httproute-%s", f.XR.Name))
	observedStructured, err := composer.ConvertObserved[gwapiv1.HTTPRoute](f.Observed, resourceName)
	if err != nil {
		return nil, err
	}

	return &HttpRoute{
		XComposer: XComposer{
			FunctionContext: f,
			ResourceName:    resourceName,
			ConditionType:   "HttpRouteReady",
		},
		ObservedResource: observedStructured,
	}, nil
}

// ComposeDesiredResource builds the desired HTTPRoute and wraps it as a
// DesiredResource for inclusion in the function response.
func (h *HttpRoute) ComposeDesiredResource() (*composer.DesiredResource, error) {
	return h.ComposeDesiredResourceFrom(h.createResource())
}

// isReady returns true if the observed HTTPRoute has been accepted by at
// least one parent gateway, as indicated by the RouteConditionAccepted
// condition being True.
func (h *HttpRoute) IsReady() bool {
	if h.ObservedResource == nil {
		return false
	}
	for _, parent := range h.ObservedResource.Status.Parents {
		for _, condition := range parent.Conditions {
			routeAccepted := string(condition.Type) == string(gwapiv1.RouteConditionAccepted)
			if routeAccepted && condition.Status == metav1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

// createResource constructs the Gateway API HTTPRoute spec from the XDeployment.
// Returns nil if either port or hostname is not set, signaling that the
// HTTPRoute should not be created.
func (h *HttpRoute) createResource() *gwapiv1.HTTPRoute {
	log := h.FunctionContext.Log
	xd := h.FunctionContext.XR

	if h.FunctionContext.Defaults == nil || h.FunctionContext.Defaults.Gateway == nil {
		log.Debug("no default values found", "name", h.XComposer.ResourceName)
		return nil
	}
	gateway := h.FunctionContext.Defaults.Gateway

	if xd.Spec.Port == nil || xd.Spec.Hostname == "" {
		log.Debug("no port or hostname specifed, skipping httpRoute", "name", h.XComposer.ResourceName)
		return nil
	}

	return &gwapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name: xd.GetName(),
		},
		Spec: gwapiv1.HTTPRouteSpec{
			CommonRouteSpec: gwapiv1.CommonRouteSpec{
				ParentRefs: []gwapiv1.ParentReference{
					{
						Name:      gwapiv1.ObjectName(gateway.Name),
						Namespace: ptr.To(gwapiv1.Namespace(gateway.Namespace)),
					},
				},
			},
			Hostnames: []gwapiv1.Hostname{
				gwapiv1.Hostname(xd.Spec.Hostname),
			},
			Rules: []gwapiv1.HTTPRouteRule{
				{
					BackendRefs: []gwapiv1.HTTPBackendRef{
						{
							BackendRef: gwapiv1.BackendRef{
								BackendObjectReference: gwapiv1.BackendObjectReference{
									Name: gwapiv1.ObjectName(xd.GetName()),
									Port: ptr.To(gwapiv1.PortNumber(8080)),
								},
							},
						},
					},
				},
			},
		},
	}
}
