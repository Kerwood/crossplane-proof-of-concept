package main

import (
	"composer"
	defaults "github.com/crossplane/function-xdeployment/input/v1beta1"
	"github.com/kerwood/crossplane-xrd-generator/resources/xdeployment"
)

// XContext is the concrete FunctionContext for this function.
type XContext = composer.FunctionContext[*xdeployment.XDeployment, *defaults.XDeploymentDefaults]

// XComposer is the concrete BaseComposer for this function.
type XComposer = composer.BaseComposer[*xdeployment.XDeployment, *defaults.XDeploymentDefaults]
