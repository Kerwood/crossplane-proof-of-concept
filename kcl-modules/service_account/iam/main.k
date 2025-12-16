import k8s_config_connector.v1beta1.iam_cnrm_cloud_google_com_v1beta1_i_a_m_policy_member as policy_member
import k8s.apimachinery.pkg.apis.meta.v1 as metav1
import k8s_config_connector.v1beta1.iam_cnrm_cloud_google_com_v1beta1_i_a_m_service_account as gsa

schema ServiceAccountConfig:
    """Configuration schema for creating a Google Service Account"""
    name: str
    namespace?: str = None
    annotations?: {str:str} = {}
    labels?: {str:str} = {}

ServiceAccount = lambda c: ServiceAccountConfig -> gsa.IAMServiceAccount {
    gsa.IAMServiceAccount {
        metadata: metav1.ObjectMeta {
            name: c.name
            labels: c.labels
            annotations: c.annotations
            namespace: c.namespace or ""
        }
        spec: gsa.IamCnrmCloudGoogleComV1beta1IAMServiceAccountSpec {
            displayName: c.name.capitalize()
            description: "Application service account for " + c.name
        }
    }
}
schema WorkloadIdentityUserPolicyConfig:
    """Configuration schema for creating a Kubernetes Service Account"""
    workloadIdentityPool: str
    namespace: str
    annotations?: {str:str} = {}
    labels?: {str:str} = {}
    gcpServiceAccountId: str

WorkloadIdentityUserPolicy = lambda c: WorkloadIdentityUserPolicyConfig -> policy_member.IAMPolicyMember {
    _sa_name = c.gcpServiceAccountId.split("@")[0]

    policy_member.IAMPolicyMember {
        metadata: metav1.ObjectMeta {
            name: _sa_name + "-workload-identity-user"
            labels: c.labels
            annotations: c.annotations
            namespace: c.namespace or ""
        }
        spec: policy_member.IamCnrmCloudGoogleComV1beta1IAMPolicyMemberSpec {
            role = "roles/iam.workloadIdentityUser"
            member = "serviceAccount:" + c.workloadIdentityPool + ".svc.id.goog[" + c.namespace + "/" + _sa_name + "]"
            resourceRef: policy_member.IamCnrmCloudGoogleComV1beta1IAMPolicyMemberSpecResourceRef {
                kind: "IAMServiceAccount"
                name: _sa_name
                apiVersion: "iam.cnrm.cloud.google.com/v1beta1"
            }
        }
    }
}
