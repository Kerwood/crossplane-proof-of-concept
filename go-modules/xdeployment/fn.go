package main

import (
	"composer"
	"context"
	"fmt"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/response"
	defaults "github.com/crossplane/function-xdeployment/input/v1beta1"
	"github.com/kerwood/crossplane-xrd-generator/resources/xdeployment"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// Function implements the Crossplane composition function gRPC service.
type Function struct {
	fnv1.UnimplementedFunctionRunnerServiceServer
	log logging.Logger
}

// Register Kubernetes resource types with the composed resource scheme.
// Required for each resource type the function composes.
// Panics on failure since scheme registration errors are unrecoverable.
func init() {
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	must(appsv1.AddToScheme(composed.Scheme))
	must(corev1.AddToScheme(composed.Scheme))
	must(gwapiv1.Install(composed.Scheme))
}

// InternalErrorResponse marks the function response as fatally failed due to an
// internal error. It sets a FunctionSuccess=False condition with reason
// InternalError on both the composite resource and claim, then records the
// error as fatal to stop the composition pipeline.
func InternalErrorResponse(rsp *fnv1.RunFunctionResponse, err error) {
	response.ConditionFalse(rsp, "FunctionSuccess", "InternalError").
		TargetCompositeAndClaim()

	response.Fatal(rsp, err)
}

// RunFunction is the entry point for the composition function. It deserializes
// the observed composite resource, composes the desired Kubernetes resources,
// and sets status conditions based on each resource's readiness.
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	f.log.Info("Running function", "tag", req.GetMeta().GetTag())

	rsp := response.To(req, response.DefaultTTL)

	// Parse the function input configuration (e.g., gateway settings)
	defaultValues := &defaults.XDeploymentDefaults{}
	if err := request.GetInput(req, defaultValues); err != nil {
		InternalErrorResponse(rsp, errors.Wrapf(err, "cannot get Function input from %T", req))
		return rsp, nil
	}

	// Get observed composed resources to check actual deployment status
	observed, err := request.GetObservedComposedResources(req)
	if err != nil {
		InternalErrorResponse(rsp, errors.Wrapf(err, "cannot get observed resources from %T", req))
		return rsp, nil
	}

	// Get all desired composed resources from the request.
	desired, err := request.GetDesiredComposedResources(req)
	if err != nil {
		InternalErrorResponse(rsp, errors.Wrapf(err, "cannot get desired resources from %T", req))
		return rsp, nil
	}

	// Get observed composite resource (XR)
	xr, err := request.GetObservedCompositeResource(req)
	if err != nil {
		InternalErrorResponse(rsp, errors.Wrapf(err, "cannot get observedf composite resources from %T", req))
		return rsp, nil
	}

	// Convert the observed composite resource to the typed XDeployment struct
	var xd xdeployment.XDeployment
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(xr.Resource.UnstructuredContent(), &xd); err != nil {
		InternalErrorResponse(rsp, errors.Wrapf(err, "cannot convert composite resource to XDeployment"))
		return rsp, nil
	}

	// Create an updated logger with useful information about the XR.
	log := f.log.WithValues(
		"xr-version", xr.Resource.GetAPIVersion(),
		"xr-kind", xr.Resource.GetKind(),
		"xr-name", xr.Resource.GetName(),
	)

	// Build the shared context passed to all resource composers
	fnContext := XContext{
		Observed:         observed,
		FunctionResponse: rsp,
		XR:               &xd,
		Defaults:         defaultValues,
		Log:              log,
	}

	// Initialize resource composers
	deployment, err := NewDeployment(fnContext)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot create deployment"))
		return rsp, nil
	}

	service, err := NewService(fnContext)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot create service"))
		return rsp, nil
	}

	httpRoute, err := NewHttpRoute(fnContext)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot create httpRoute"))
		return rsp, nil
	}

	// Collect all resources to compose
	resources := []composer.ComposableResource{
		deployment,
		service,
		httpRoute,
	}

	// Compose each resource and set its readiness condition
	for _, r := range resources {
		desiredResource, err := r.ComposeDesiredResource()
		if err != nil {
			response.ConditionFalse(rsp, r.GetConditionType(), "CompositionError").
				WithMessage(err.Error()).
				TargetComposite()
			return nil, err
		}

		if desiredResource == nil {
			continue
		}

		if r.IsReady() {
			desiredResource.Resource.Ready = resource.ReadyTrue
			response.ConditionTrue(rsp, r.GetConditionType(), "Available").
				TargetComposite()
		} else {
			response.ConditionFalse(rsp, r.GetConditionType(), "Unavailable").
				WithMessage(fmt.Sprintf("%s is not yet available", r.GetConditionType())).
				TargetComposite()
		}

		log.Info("Added desired resource", "name", desiredResource.Name)
		desired[desiredResource.Name] = desiredResource.Resource
	}

	// Set the composed resources on the response
	if err := response.SetDesiredComposedResources(rsp, desired); err != nil {
		InternalErrorResponse(rsp, errors.Wrapf(err, "cannot set desired composed resources in %T", rsp))
		return rsp, nil
	}

	return rsp, nil
}
