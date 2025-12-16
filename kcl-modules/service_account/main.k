import k8s.apimachinery.pkg.apis.meta.v1 as metav1
import k8s.api.core.v1 as corev1

schema ServiceAccountConfig:
    """Configuration schema for creating a Kubernetes Service Account"""
    name: str
    namespace?: str = None
    annotations?: {str:str} = {}
    labels?: {str:str} = {}
    gcpServiceAccountId?: str

ServiceAccount = lambda c: ServiceAccountConfig -> corev1.ServiceAccount {
    if c.gcpServiceAccountId:
        c.annotations["iam.gke.io/gcp-service-account"] = c.gcpServiceAccountId

    corev1.ServiceAccount {
        metadata: metav1.ObjectMeta {
            name: c.name
            labels: c.labels
            annotations: c.annotations
            namespace: c.namespace or ""
        }
    }
}
