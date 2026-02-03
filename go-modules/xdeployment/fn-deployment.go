package main

import (
	"fmt"
	"sort"

	"composer"
	"github.com/crossplane/function-sdk-go/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// Deployment composes a Kubernetes Deployment from the XDeployment spec.
// It manages the desired Deployment resource and checks the observed
// Deployment's availability condition to determine readiness.
type Deployment struct {
	XComposer
	ObservedResource *appsv1.Deployment
}

// NewDeployment creates a new Deployment composer. It looks up the observed
// Deployment resource and deserializes it for readiness checks. Returns an
// error if the observed resource exists but cannot be deserialized.
func NewDeployment(f XContext) (*Deployment, error) {
	resourceName := resource.Name(fmt.Sprintf("xdeployment-deployment-%s", f.XR.Name))
	observedStructured, err := composer.ConvertObserved[appsv1.Deployment](f.Observed, resourceName)
	if err != nil {
		return nil, err
	}

	return &Deployment{
		XComposer: XComposer{
			FunctionContext: f,
			ResourceName:    resourceName,
			ConditionType:   "DeploymentReady",
		},
		ObservedResource: observedStructured,
	}, nil
}

// ComposeDesiredResource builds the desired Deployment and wraps it as a
// DesiredResource for inclusion in the function response.
func (d *Deployment) ComposeDesiredResource() (*composer.DesiredResource, error) {
	return d.ComposeDesiredResourceFrom(d.createResource())
}

// isReady returns true if the observed Deployment has the Available condition
// set to True, indicating that the minimum number of replicas are running.
func (d *Deployment) IsReady() bool {
	if d.ObservedResource == nil {
		return false
	}
	for _, condition := range d.ObservedResource.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable &&
			condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// createResource constructs the Kubernetes Deployment spec from the XDeployment.
// It sets a default replica count of 2 if not specified, configures the
// container with the specified image and environment variables, and optionally
// exposes a container port if one is defined.
func (d *Deployment) createResource() *appsv1.Deployment {
	log := d.FunctionContext.Log
	xd := d.FunctionContext.XR

	replicas := xd.Spec.Replicas
	if replicas == nil {
		log.Debug("replicas not set on deployment, defaulting to 2", "name", d.XComposer.ResourceName)
		replicas = ptr.To(int32(2))
	}

	container := corev1.Container{
		Name:  xd.GetName(),
		Image: xd.Spec.Image,
		Env:   convertEnvVars(xd.Spec.Env),
	}

	if xd.Spec.Port != nil {
		container.Ports = []corev1.ContainerPort{{
			Name:          "http",
			ContainerPort: int32(*xd.Spec.Port),
		}}
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: xd.GetName(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": xd.GetName(),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: xd.GetName(),
					Labels: map[string]string{
						"app.kubernetes.io/name": xd.GetName(),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
		},
	}

}

// convertEnvVars converts a map of environment variables to a sorted slice of
// corev1.EnvVar. The keys are sorted alphabetically to ensure deterministic
// ordering across reconciliation loops.
func convertEnvVars(envMap map[string]string) []corev1.EnvVar {
	if len(envMap) == 0 {
		return nil
	}

	// Sort keys for consistent ordering
	keys := make([]string, 0, len(envMap))
	for k := range envMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]corev1.EnvVar, 0, len(envMap))
	for _, key := range keys {
		result = append(result, corev1.EnvVar{
			Name:  key,
			Value: envMap[key],
		})
	}

	return result
}
