package main

import (
	"composer"

	defaults "github.com/crossplane/function-xappregistration/input/v1beta1"
	"github.com/kerwood/crossplane-xrd-generator/resources/xappregistration"
)

// XContext is the concrete FunctionContext for this function.
type XContext = composer.FunctionContext[*xappregistration.XAppRegistration, *defaults.XAppRegistrationDefaults]

// XComposer is the concrete BaseComposer for this function.
type XComposer = composer.BaseComposer[*xappregistration.XAppRegistration, *defaults.XAppRegistrationDefaults]
