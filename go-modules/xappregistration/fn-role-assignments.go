package main

import (
	"composer"
	"fmt"
	commonv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	commonv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	"github.com/crossplane/function-sdk-go/resource"
	appv1beta1 "github.com/upbound/provider-azuread/v2/apis/namespaced/app/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RoleAssignment composes a Kubernetes RoleAssignment from the XAppRegistration spec.
type RoleAssignment struct {
	XComposer
	ObservedResource *appv1beta1.RoleAssignment
	GroupUID         *string
	AppRoleUID       *string
	Identifier       *string
}

// NewApplication creates a new RoleAssignment composer. It looks up the observed RoleAssignment
// resource and deserializes it for readiness check. Returns an error if the observed resource exists but cannot be deserialized.
func NewRoleAssignments(f XContext) ([]composer.ComposableResource, error) {
	roleAssignments := []composer.ComposableResource{}

	for _, ra := range f.XR.Spec.RoleAssignments {
		uuid := generateRoleUID(f.XR.GetName() + ra.RoleName)
		for _, group := range ra.AssignedGroups {
			identifier := hashString(uuid + group)
			resourceName := resource.Name(fmt.Sprintf("xappregistration-role-assignment-%s-%s", f.XR.Name, identifier))
			observedStructured, err := composer.ConvertObserved[appv1beta1.RoleAssignment](f.Observed, resourceName)
			if err != nil {
				return nil, err
			}

			ra := &RoleAssignment{
				XComposer: XComposer{
					FunctionContext: f,
					ResourceName:    resourceName,
					ConditionType:   "RoleAssignment(" + identifier + ")Ready",
				},
				ObservedResource: observedStructured,
				GroupUID:         &group,
				AppRoleUID:       &uuid,
				Identifier:       &identifier,
			}

			roleAssignments = append(roleAssignments, ra)
		}
	}

	return roleAssignments, nil
}

// ComposeDesiredResource builds the desired RoleAssignment and wraps it as a
// DesiredResource for inclusion in the function response.
func (r *RoleAssignment) ComposeDesiredResource() (*composer.DesiredResource, error) {
	return r.ComposeDesiredResourceFrom(r.CreateResource())
}

// IsReady returns true if the observed resource has a Ready condition with status True.
func (r *RoleAssignment) IsReady() bool {
	if r.ObservedResource == nil {
		return false
	}
	for _, c := range r.ObservedResource.Status.Conditions {
		if c.Type == "Ready" && c.Status == "True" {
			return true
		}
	}
	return false
}

// CreateResource constructs the Kubernetes RoleAssignment spec from the XAppRegistration.
func (r *RoleAssignment) CreateResource() *appv1beta1.RoleAssignment {
	xr := r.FunctionContext.XR
	defaults := r.FunctionContext.Defaults
	name := xr.GetName()

	return &appv1beta1.RoleAssignment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name + "-" + *r.Identifier,
		},
		Spec: appv1beta1.RoleAssignmentSpec{
			ForProvider: appv1beta1.RoleAssignmentParameters{
				AppRoleID: r.AppRoleUID,
				ResourceObjectIDRef: &commonv1.NamespacedReference{
					Name: name,
				},
				PrincipalObjectID: r.GroupUID,
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
