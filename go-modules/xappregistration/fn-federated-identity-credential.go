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

// FederatedIdentityCredential composes a Kubernetes FederatedIdentityCredential from the XAppRegistration spec.
type FederatedIdentityCredential struct {
	XComposer
	ObservedResource *applicationsv1beta1.FederatedIdentityCredential
}

// NewFederatedIdentityCredential creates a new FederatedIdentityCredential composer. It looks up the observed FederatedIdentityCredential
// resource and deserializes it for readiness check. Returns an error if the observed resource exists but cannot be deserialized.
func NewFederatedIdentityCredential(f XContext) (composer.ComposableResource, error) {
	resourceName := resource.Name(fmt.Sprintf("xappregistration-federated-identity-credential-%s", f.XR.Name))
	observedStructured, err := composer.ConvertObserved[applicationsv1beta1.FederatedIdentityCredential](f.Observed, resourceName)
	if err != nil {
		return nil, err
	}

	return &FederatedIdentityCredential{
		XComposer: XComposer{
			FunctionContext: f,
			ResourceName:    resourceName,
			ConditionType:   "FederatedIdentityCredentialReady",
		},
		ObservedResource: observedStructured,
	}, nil
}

// ComposeDesiredResource builds the desired FederatedIdentityCredential and wraps it as a
// DesiredResource for inclusion in the function response.
func (f *FederatedIdentityCredential) ComposeDesiredResource() (*composer.DesiredResource, error) {
	return f.ComposeDesiredResourceFrom(f.CreateResource())
}

// IsReady returns true if the observed resource has a Ready condition with status True.
func (f *FederatedIdentityCredential) IsReady() bool {
	if f.ObservedResource == nil {
		return false
	}
	for _, c := range f.ObservedResource.Status.Conditions {
		if c.Type == "Ready" && c.Status == "True" {
			return true
		}
	}
	return false
}

// CreateResource constructs the Kubernetes FederatedIdentityCredential spec from the XAppRegistration.
func (f *FederatedIdentityCredential) CreateResource() *applicationsv1beta1.FederatedIdentityCredential {
	xr := f.FunctionContext.XR
	defaults := f.FunctionContext.Defaults

	displayName := xr.GetName()
	namePrefix := defaults.AppRegNamePrefix
	if namePrefix != "" {
		displayName = namePrefix + " - " + displayName
	}

	return &applicationsv1beta1.FederatedIdentityCredential{
		ObjectMeta: metav1.ObjectMeta{
			Name: xr.GetName(),
		},
		Spec: applicationsv1beta1.FederatedIdentityCredentialSpec{
			ForProvider: applicationsv1beta1.FederatedIdentityCredentialParameters{
				DisplayName: ptr.To("Kubernetes-TokenRequest-API"),
				ApplicationIDRef: &commonv1.NamespacedReference{
					Name: xr.GetName(),
				},
				Audiences: []*string{
					ptr.To("api://AzureADTokenExchange"),
				},
				Issuer:  &defaults.ClusterIssuerUrl,
				Subject: ptr.To(fmt.Sprintf("system:serviceaccount:%s:%s", xr.GetNamespace(), xr.Spec.ServiceAccountName)),
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
