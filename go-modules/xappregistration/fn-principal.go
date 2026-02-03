package main

import (
	"fmt"

	"composer"

	commonv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	commonv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	"github.com/crossplane/function-sdk-go/resource"
	serviceprincipalsv1beta1 "github.com/upbound/provider-azuread/v2/apis/namespaced/serviceprincipals/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// Principal composes a Kubernetes Principal from the XAppRegistration spec.
type Principal struct {
	XComposer
	ObservedResource *serviceprincipalsv1beta1.Principal
}

// NewPrincipal creates a new Principal composer. It looks up the observed Principal
// resource and deserializes it for readiness check. Returns an error if the
// observed resource exists but cannot be deserialized.
func NewPrincipal(f XContext) (composer.ComposableResource, error) {
	resourceName := resource.Name(fmt.Sprintf("xappregistration-principal-%s", f.XR.Name))
	observedStructured, err := composer.ConvertObserved[serviceprincipalsv1beta1.Principal](f.Observed, resourceName)
	if err != nil {
		return nil, err
	}

	return &Principal{
		XComposer: XComposer{
			FunctionContext: f,
			ResourceName:    resourceName,
			ConditionType:   "PrincipalReady",
		},
		ObservedResource: observedStructured,
	}, nil
}

// ComposeDesiredResource builds the desired Principal and wraps it as a
// DesiredResource for inclusion in the function response.
func (p *Principal) ComposeDesiredResource() (*composer.DesiredResource, error) {
	return p.ComposeDesiredResourceFrom(p.CreateResource())
}

// IsReady returns true if the observed resource has a Ready condition with status True.
func (p *Principal) IsReady() bool {
	if p.ObservedResource == nil {
		return false
	}
	for _, c := range p.ObservedResource.Status.Conditions {
		if c.Type == "Ready" && c.Status == "True" {
			return true
		}
	}
	return false
}

func (p *Principal) GetConnectionDetails() map[string]string {
	if p.ObservedResource == nil {
		return nil
	}
	tenantId := p.ObservedResource.Status.AtProvider.ApplicationTenantID
	clientId := p.ObservedResource.Status.AtProvider.ClientID

	if tenantId == nil || clientId == nil {
		return nil
	}

	return map[string]string{
		"AZURE_CLIENT_ID":    *clientId,
		"AZURE_TENANT_ID":    *tenantId,
		"AZURE_ISSUER_URL":   "https://login.microsoftonline.com/" + *tenantId + "/v2.0",
		"AZURE_CLIENT_SCOPE": *clientId + "/.default",
	}
}

// CreateResource constructs the Kubernetes Principal spec from the XR.
func (p *Principal) CreateResource() *serviceprincipalsv1beta1.Principal {
	xr := p.FunctionContext.XR
	defaults := p.FunctionContext.Defaults

	assignmentRequired := false
	if len(xr.Spec.RoleAssignments) > 0 {
		assignmentRequired = true
	}

	return &serviceprincipalsv1beta1.Principal{
		ObjectMeta: metav1.ObjectMeta{
			Name: xr.GetName(),
		},
		Spec: serviceprincipalsv1beta1.PrincipalSpec{
			ForProvider: serviceprincipalsv1beta1.PrincipalParameters{
				Owners: []*string{
					&defaults.CrossplaneServicePrincipalID,
				},
				AppRoleAssignmentRequired: &assignmentRequired,
				FeatureTags: []serviceprincipalsv1beta1.FeatureTagsParameters{{
					Enterprise: ptr.To(true),
				}},
				// UseExisting: ptr.To(true),
				ClientIDRef: &commonv1.NamespacedReference{
					Name: xr.GetName(),
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
