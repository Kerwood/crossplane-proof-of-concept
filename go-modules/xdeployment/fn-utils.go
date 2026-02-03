package main

import (
	"github.com/kerwood/crossplane-xrd-generator/resources/xdeployment"
)

func OAuth2ProxyEnabled(xr *xdeployment.XDeployment) bool {
	sso := xr.Spec.SingleSignOn
	return sso != nil && sso.EnableAuthProxy
}

func ConnectionDetailsAvailable(xr *xdeployment.XDeployment) bool {
	sso := xr.Spec.SingleSignOn
	return sso != nil && sso.ConnectionDetailsSecretRef != nil
}
