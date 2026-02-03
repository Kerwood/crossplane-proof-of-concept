package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"

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
	// log := d.FunctionContext.Log
	xr := d.FunctionContext.XR

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: xr.GetName(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(ptr.Deref(xr.Spec.Replicas, int32(2))),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": xr.GetName(),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: xr.GetName(),
					Labels: map[string]string{
						"app.kubernetes.io/name": xr.GetName(),
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{},
					Containers: []corev1.Container{
						{
							Name:    xr.GetName(),
							Image:   xr.Spec.Image + ":" + xr.Spec.Tag,
							Env:     convertEnvVars(xr.Spec.Env),
							EnvFrom: []corev1.EnvFromSource{},
						}},
					ServiceAccountName: ptr.Deref(xr.Spec.ServiceAccountName, ""),
					Volumes:            []corev1.Volume{},
				},
			},
		},
	}

	if ConnectionDetailsAvailable(xr) {
		envFrom := corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: d.FunctionContext.XR.Spec.SingleSignOn.ConnectionDetailsSecretRef.Name,
				},
			},
		}
		deployment.Spec.Template.Spec.Containers[0].EnvFrom = append(
			deployment.Spec.Template.Spec.Containers[0].EnvFrom, envFrom,
		)
	}

	if xr.Spec.Port != nil {
		port := int32(*xr.Spec.Port)
		if OAuth2ProxyEnabled(xr) && ConnectionDetailsAvailable(xr) {
			port = 4180
		}
		deployment.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{{
			Name:          "http",
			ContainerPort: port,
		}}
	}

	if OAuth2ProxyEnabled(xr) && ConnectionDetailsAvailable(xr) {
		container, volume := OAuth2ProxyComponents(&d.FunctionContext)
		deployment.Spec.Template.Spec.InitContainers = append(
			deployment.Spec.Template.Spec.InitContainers, container,
		)
		deployment.Spec.Template.Spec.Volumes = append(
			deployment.Spec.Template.Spec.Volumes, volume,
		)
	}

	return deployment
}

func OAuth2ProxyComponents(f *XContext) (corev1.Container, corev1.Volume) {
	defaults := f.Defaults.OAuth2ProxyConfig
	image := defaults.Image + ":" + defaults.Tag

	container := corev1.Container{
		Name:          "oauth2proxy",
		Image:         image,
		RestartPolicy: ptr.To(corev1.ContainerRestartPolicyAlways),
		Ports: []corev1.ContainerPort{{
			Name:          "http",
			ContainerPort: int32(4180),
		}},
		Args: []string{
			"--oidc-issuer-url=" + defaults.IssuerURL,
			"--entra-id-federated-token-auth=true",
			"--cookie-domain=" + defaults.CookieDomain,
			"--http-address=0.0.0.0:4180",
			"--upstream=http://127.0.0.1:" + strconv.Itoa(int(ptr.Deref(f.XR.Spec.Port, 8080))),
			"--provider=entra-id",
			"--email-domain=*",
			"--skip-provider-button",
			"--cookie-secret=" + GenerateCookieSecret(f),
			"--scope=openid email profile",
		},
		Env: []corev1.EnvVar{
			{
				Name: "OAUTH2_PROXY_CLIENT_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: f.XR.Spec.SingleSignOn.ConnectionDetailsSecretRef.Name,
						},
						Key: "AZURE_CLIENT_ID",
					},
				},
			},
			{
				Name:  "AZURE_FEDERATED_TOKEN_FILE",
				Value: "/var/run/secrets/tokens/azure-token",
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/var/run/secrets/tokens/",
				Name:      "azure-token",
				ReadOnly:  true,
			},
		},
	}

	volume := corev1.Volume{
		Name: "azure-token",
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
							Audience:          "api://AzureADTokenExchange",
							ExpirationSeconds: ptr.To(int64(3600)),
							Path:              "azure-token",
						},
					},
				},
			},
		},
	}

	return container, volume
}

func GenerateCookieSecret(f *XContext) string {
	seed := f.Defaults.OAuth2ProxyConfig.CookieSecretSeed + f.XR.Name
	hash := sha256.Sum256([]byte(seed))
	encoded := base64.StdEncoding.EncodeToString(hash[:])
	return encoded[:32]
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
