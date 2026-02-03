// Package v1beta1 contains the input type for this Function
// +kubebuilder:object:generate=true
// +groupName=defaults.fn.crossplane.io
// +versionName=v1beta1
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This isn't a custom resource, in the sense that we never install its CRD.
// It is a KRM-like object, so we generate a CRD to describe its schema.

// TODO: Add your input type here! It doesn't need to be called 'Input', you can
// rename it to anything you like.

// Input can be used to provide input to this Function.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=crossplane
type XDeploymentDefaults struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Gateway configures the parent Gateway reference for HTTPRoutes.
	// Required when using HTTPRoute resources.
	Gateway           *GatewayConfig     `json:"gateway,omitempty"`
	OAuth2ProxyConfig *OAuth2ProxyConfig `json:"oauth2ProxyConfig,omitempty"`
}

type GatewayConfig struct {
	// Parent Gateway resource name
	Name string `json:"name"`

	// Parent Gateway resource namespace
	Namespace string `json:"namespace"`
}

type OAuth2ProxyConfig struct {
	Image            string `json:"image" required:"true"`
	Tag              string `json:"tag" required:"true"`
	IssuerURL        string `json:"issuerUrl" required:"true"`
	CookieDomain     string `json:"cookieDomain" required:"true"`
	CookieSecretSeed string `json:"cookieSecretSeed" required:"true"`
}
