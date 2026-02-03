package main

import (
	"fmt"

	"composer"
	commonv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	commonv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	"github.com/crossplane/function-sdk-go/resource"
	applicationsv1beta1 "github.com/upbound/provider-azuread/v2/apis/namespaced/applications/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// Application composes a Kubernetes Application from the XAppRegistration spec.
type Application struct {
	XComposer
	ObservedResource *applicationsv1beta1.Application
}

// NewApplication creates a new Application composer. It looks up the observed Application
// resource and deserializes it for readiness check. Returns an error if the
// observed resource exists but cannot be deserialized.
func NewApplication(f XContext) (composer.ComposableResource, error) {
	resourceName := resource.Name(fmt.Sprintf("xappregistration-application-%s", f.XR.Name))
	observedStructured, err := composer.ConvertObserved[applicationsv1beta1.Application](f.Observed, resourceName)
	if err != nil {
		return nil, err
	}

	return &Application{
		XComposer: XComposer{
			FunctionContext: f,
			ResourceName:    resourceName,
			ConditionType:   "ApplicationReady",
		},
		ObservedResource: observedStructured,
	}, nil
}

// ComposeDesiredResource builds the desired Application and wraps it as a
// DesiredResource for inclusion in the function response.
func (s *Application) ComposeDesiredResource() (*composer.DesiredResource, error) {
	return s.ComposeDesiredResourceFrom(s.CreateResource())
}

// IsReady returns true if the observed resource has a Ready condition with status True.
func (s *Application) IsReady() bool {
	if s.ObservedResource == nil {
		return false
	}
	for _, c := range s.ObservedResource.Status.Conditions {
		if c.Type == "Ready" && c.Status == "True" {
			return true
		}
	}
	return false
}

// CreateResource constructs the Kubernetes Application spec from the XAppRegistration.
func (s *Application) CreateResource() *applicationsv1beta1.Application {
	xr := s.FunctionContext.XR
	defaults := s.FunctionContext.Defaults

	displayName := xr.GetName()
	namePrefix := defaults.AppRegNamePrefix
	if namePrefix != "" {
		displayName = namePrefix + " - " + displayName
	}

	return &applicationsv1beta1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name: xr.GetName(),
		},
		Spec: applicationsv1beta1.ApplicationSpec{
			ForProvider: applicationsv1beta1.ApplicationParameters{
				DisplayName: ptr.To(displayName),
				Owners: []*string{
					&defaults.CrossplaneServicePrincipalID,
				},
				Web: &applicationsv1beta1.WebParameters{
					RedirectUris: ToSliceOfPtrs(xr.Spec.RedirectURLs),
				},
				API: &applicationsv1beta1.APIParameters{
					RequestedAccessTokenVersion: ptr.To(float64(2)),
				},
			},
			ManagedResourceSpec: commonv2.ManagedResourceSpec{
				ProviderConfigReference: &commonv1.ProviderConfigReference{
					Name: defaults.ProviderConfigRef.Name,
					Kind: defaults.ProviderConfigRef.Kind,
				},
			},
		},
	}
}
