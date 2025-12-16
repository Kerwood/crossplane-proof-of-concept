import k8s_config_connector.v1beta1.storage_cnrm_cloud_google_com_v1beta1_storage_bucket as sb
import k8s_config_connector.v1beta1.iam_cnrm_cloud_google_com_v1beta1_i_a_m_policy_member as policy_member
import k8s.apimachinery.pkg.apis.meta.v1 as metav1

schema StorageBucketConfig:
    """Configuration schema for creating a Google Storage Bucket"""
    name: str
    namespace?: str = None
    annotations?: {str:str} = {}
    labels?: {str:str} = {}

StorageBucket = lambda c: StorageBucketConfig -> sb.StorageBucket {
    sb.StorageBucket {
        metadata: metav1.ObjectMeta {
            name: c.name
            labels: c.labels
            annotations: c.annotations
            namespace: c.namespace or ""
        }
        spec: sb.StorageCnrmCloudGoogleComV1beta1StorageBucketSpec {
            storageClass: "STANDARD"
            location: "europe-west3"
            uniformBucketLevelAccess: True
        }
    }
}

schema SetObjectViewerConfig:
    gcpServiceAccountId: str
    storageBucketName: str
    namespace?: str = None
    annotations?: {str:str} = {}
    labels?: {str:str} = {}

SetObjectViewer = lambda c: SetObjectViewerConfig -> policy_member.IAMPolicyMember {
    _sa_name = c.gcpServiceAccountId.split("@")[0]

    policy_member.IAMPolicyMember {
        metadata: metav1.ObjectMeta {
            name: _sa_name + "-storage-bucket-object-viewer"
            labels: c.labels
            annotations: c.annotations
            namespace: c.namespace or ""
        }
        spec: policy_member.IamCnrmCloudGoogleComV1beta1IAMPolicyMemberSpec {
            role = "roles/storage.objectViewer"
            member = "serviceAccount:" + c.gcpServiceAccountId
            resourceRef: policy_member.IamCnrmCloudGoogleComV1beta1IAMPolicyMemberSpecResourceRef {
                kind: "StorageBucket"
                name: c.storageBucketName
                apiVersion: "storage.cnrm.cloud.google.com/v1beta1"
            }
        }
    }
}
