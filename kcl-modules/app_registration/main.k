import k8s.apimachinery.pkg.apis.meta.v1 as metav1

schema AppRegistrationConfig:
    """Configuration schema for creating an Entra ID App Registration"""
    name: str
    issuer: str
    namespace: str
    serviceAccountName: str
    providerConfigRef: ProviderConfigRef
    redirectUris?: [str] = []
    applicationAnnotations?: {str:str} = {}
    principalAnnotations?: {str:str} = {}
    federatedIdentityAnnotations?: {str:str} = {}
    labels?: {str:str} = {}

schema AppRegistrationBundle:
    application: Application
    servicePrincipal: Principal
    federatedIdentity: FederatedIdentityCredential

schema ProviderConfigRef:
    kind: str
    name: str

AppRegistration = lambda c: AppRegistrationConfig -> AppRegistrationBundle {
    _app = Application {
        metadata: metav1.ObjectMeta {
            name: c.name
            labels = c.labels
            annotations: c.applicationAnnotations
        }
        spec: {
            forProvider: {
                displayName: c.name
                web: {
                    redirectUris: c.redirectUris
                }
            }
            providerConfigRef: {
                **c.providerConfigRef
            }
        }
    }

    _sp = Principal {
        metadata: metav1.ObjectMeta {
            name: c.name
            labels = c.labels
            annotations: c.principalAnnotations
        }
        spec: {
            forProvider: {
                featureTags: [{
                    enterprise: True
                }]
                useExisting: True
                clientIdRef: {
                    name: c.name
                }
            }
            providerConfigRef: {
                **c.providerConfigRef
            }
        }
    }

    _fic = FederatedIdentityCredential {
        metadata: metav1.ObjectMeta {
            name: c.name
            labels = c.labels
            annotations: c.federatedIdentityAnnotations
        }
        spec: {
            forProvider: {
                applicationIdRef: {
                    name: c.name
                }
                audiences: ["api://AzureADTokenExchange"]
                displayName: "kubernetes-workload-identity"
                subject: "system:serviceaccount:" + c.namespace + ":" + c.serviceAccountName
                issuer: c.issuer
            }
            providerConfigRef: {
                **c.providerConfigRef
            }
        }
    }

    AppRegistrationBundle {
        application: _app
        servicePrincipal: _sp
        federatedIdentity: _fic
    }
}
