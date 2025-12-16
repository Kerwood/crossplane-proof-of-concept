import k8s.api.apps.v1 as appsv1
import k8s.api.core.v1 as corev1
import gateway_api.v1 as netv1
import k8s.apimachinery.pkg.apis.meta.v1 as metav1

# Schema for deployment configuration
schema DeploymentConfig:
    """Configuration schema for creating a Kubernetes Deployment"""
    name: str
    namespace?: str = "default"
    replicas?: int = 1
    image: str
    port?: int = None
    hostname?: str = None
    gatewayRef: netv1.GatewayNetworkingK8sIoV1HTTPRouteSpecParentRefsItems0
    deploymentAnnotations?: {str:str} = {}
    serviceAnnotations?: {str:str} = {}
    httpRouteAnnotations?: {str:str} = {}
    labels?: {str:str} = {}
    envs?: {str:str} = {}
    serviceAccountName?: str = ""

schema DeploymentBundle:
    deployment: appsv1.Deployment
    service?: corev1.Service
    httpRoute?: netv1.HTTPRoute

# Function to create a Deployment from DeploymentConfig
Deployment = lambda c: DeploymentConfig -> DeploymentBundle {
    _matchLabels = {"app.kubernetes.io/name": c.name}

    # Merge user labels with default selector labels
    _labels = {
        **_matchLabels
        **c.labels
    }

    _deployment = appsv1.Deployment {
        metadata: metav1.ObjectMeta {
            name: c.name
            labels = _labels
            annotations: c.deploymentAnnotations
        }
        spec: appsv1.DeploymentSpec {
            replicas: c.replicas
            selector: metav1.LabelSelector {
                matchLabels: _matchLabels
            }
            template: corev1.PodTemplateSpec {
                metadata: metav1.ObjectMeta {
                    labels: _matchLabels
                }
                spec: corev1.PodSpec {
                    serviceAccountName: c.serviceAccountName
                    containers: [
                        corev1.Container {
                            name: "main"
                            image: c.image
                            env: [corev1.EnvVar {
                                name: key
                                value: c.envs[key]
                            } for key in c.envs] if c.envs else []
                            ports = [
                                corev1.ContainerPort {
                                    containerPort = c.port
                                }
                            ] if c.port != None else []
                            volumeMounts: [
                                corev1.VolumeMount {
                                    name: "azure-token"
                                    mountPath: "/var/run/secrets/tokens/"
                                    readOnly: True
                                }
                            ]
                        }
                    ]
                    volumes: [
                        corev1.Volume {
                            name: "azure-token"
                            projected: corev1.ProjectedVolumeSource {
                                sources: [
                                    corev1.VolumeProjection {
                                        serviceAccountToken: corev1.ServiceAccountTokenProjection {
                                            audience: "api://AzureADTokenExchange"
                                            path: "azure-token"
                                            expirationSeconds: 3600
                                        }
                                    }
                                ]
                            }
                        }
                    ]
                }
            }
        }
    }

    _service = corev1.Service {
        metadata: metav1.ObjectMeta {
            name: c.name
            labels = _labels
            annotations: c.serviceAnnotations
        }
        spec: corev1.ServiceSpec {
            selector: _matchLabels
            ports: [
                corev1.ServicePort {
                    port: c.port
                    targetPort: c.port
                }
            ]
        }
    } if c.port != None else None

    _httpRoute = netv1.HTTPRoute {
        metadata: metav1.ObjectMeta {
            name: c.name
            labels = _labels
            annotations: c.httpRouteAnnotations
        }
        spec: netv1.GatewayNetworkingK8sIoV1HTTPRouteSpec {
            hostnames: [c.hostname]
            parentRefs: [c.gatewayRef]
            rules: [netv1.GatewayNetworkingK8sIoV1HTTPRouteSpecRulesItems0 {
                backendRefs: [netv1.GatewayNetworkingK8sIoV1HTTPRouteSpecRulesItems0BackendRefsItems0 {
                    name: c.name
                    port: c.port
                }]
                matches: [netv1.GatewayNetworkingK8sIoV1HTTPRouteSpecRulesItems0MatchesItems0 {
                    path: netv1.GatewayNetworkingK8sIoV1HTTPRouteSpecRulesItems0MatchesItems0Path {
                        type: "PathPrefix"
                        value: "/"
                    }
                }]
            }]
        }
    } if c.hostname != None and _service != None else None

    DeploymentBundle {
        deployment: _deployment
        service: _service
        httpRoute: _httpRoute
    }
}
