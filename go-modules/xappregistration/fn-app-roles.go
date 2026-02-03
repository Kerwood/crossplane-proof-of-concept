package main

import (
	"fmt"

	"composer"
	"github.com/crossplane/crossplane-runtime/v2/apis/common"
	commonv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	commonv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/gosimple/slug"
	applicationsv1beta1 "github.com/upbound/provider-azuread/v2/apis/namespaced/applications/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// AppRole composes a Kubernetes AppRole from the XAppRegistration spec.
type AppRole struct {
	XComposer
	ObservedResource *applicationsv1beta1.AppRole
	RoleName         string
	RoleDescription  string
}

// NewApplication creates a new AppRole composer. It looks up the observed AppRole
// resource and deserializes it for readiness check. Returns an error if the
// observed resource exists but cannot be deserialized.
func NewAppRoles(f XContext) ([]composer.ComposableResource, error) {
	appRoles := []composer.ComposableResource{}

	for _, ra := range f.XR.Spec.RoleAssignments {
		identifier := slug.Make(ra.RoleName)
		resourceName := resource.Name(fmt.Sprintf("xappregistration-app-role-%s-%s", f.XR.Name, identifier))
		observedStructured, err := composer.ConvertObserved[applicationsv1beta1.AppRole](f.Observed, resourceName)
		if err != nil {
			return nil, err
		}

		ar := &AppRole{
			XComposer: XComposer{
				FunctionContext: f,
				ResourceName:    resourceName,
				ConditionType:   "AppRole(" + ra.RoleName + ")Ready",
			},
			ObservedResource: observedStructured,
			RoleName:         ra.RoleName,
			RoleDescription:  ra.Description,
		}

		appRoles = append(appRoles, ar)
	}

	return appRoles, nil

}

// ComposeDesiredResource builds the desired AppRole and wraps it as a
// DesiredResource for inclusion in the function response.
func (a *AppRole) ComposeDesiredResource() (*composer.DesiredResource, error) {
	return a.ComposeDesiredResourceFrom(a.CreateResource())
}

// IsReady returns true if the observed resource has a Ready condition with status True.
func (a *AppRole) IsReady() bool {
	if a.ObservedResource == nil {
		return false
	}
	for _, c := range a.ObservedResource.Status.Conditions {
		if c.Type == "Ready" && c.Status == "True" {
			return true
		}
	}
	return false
}

// CreateResource constructs the Kubernetes AppRole spec from the XAppRegistration.
func (a *AppRole) CreateResource() *applicationsv1beta1.AppRole {
	xr := a.FunctionContext.XR
	defaults := a.FunctionContext.Defaults
	uuid := generateRoleUID(xr.GetName() + a.RoleName)

	return &applicationsv1beta1.AppRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: xr.GetName() + "-" + slug.Make(a.RoleName),
		},
		Spec: applicationsv1beta1.AppRoleSpec{
			ForProvider: applicationsv1beta1.AppRoleParameters_2{
				AllowedMemberTypes: []*string{
					ptr.To("User"),
				},
				ApplicationIDRef: &common.NamespacedReference{
					Name: xr.GetName(),
				},
				DisplayName: &a.RoleName,
				Description: &a.RoleDescription,
				Value:       &a.RoleName,
				RoleID:      &uuid,
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
