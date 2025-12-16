oci_reg := "docker.io/kerwood"

[private]
default:
	@just --list

# Deploy the Azure AD Provider.
[group('Azure AD Provider')]
deploy-provider-azuread:
  kubectl apply -f ./crossplane/provider-azuread/provider.yaml
  kubectl apply -f ./crossplane/provider-azuread/deployment-runtime-config.yaml
  kubectl wait --for=condition=Healthy=True provider.pkg.crossplane.io/provider-azuread --timeout=180s
  kubectl apply -f ./crossplane/provider-azuread/cluster-provider-config.yaml

# Delete the Azure AD Provider
[group('Azure AD Provider')]
delete-provider-azuread:
  kubectl delete -f ./crossplane/provider-azuread/provider.yaml
  kubectl delete -f ./crossplane/provider-azuread/deployment-runtime-config.yaml
  kubectl delete -f ./crossplane/provider-azuread/cluster-provider-config.yaml

# Deploy the crossplane-contrib/function-kcl Function
[group('KCL Function')]
deploy-kcl-function:
  kubectl apply -f ./crossplane/function-kcl.yaml

# Delete the crossplane-contrib/function-kcl Function
[group('KCL Function')]
delete-kcl-function:
  kubectl delete -f ./crossplane/function-kcl.yaml

# Deploy additional permissions for Crossplane
[group('Additional Permissions')]
deploy-additional-xp-permissions:
  kubectl apply -f ./crossplane/additional-cluster-role.yaml

# Delete additional permissions for Crossplane
[group('Additional Permissions')]
delete-additional-xp-permissions:
  kubectl delete -f ./crossplane/additional-cluster-role.yaml

# Deploy all example-1 resources
[group('example-1')]
deploy-example-1:
  kubectl apply -f ./crossplane/example-1/app-xrd.yaml
  kubectl apply -f ./crossplane/example-1/app-composition.yaml
  kubectl apply -f ./crossplane/example-1/app-xr.yaml

# Delete all example-1 resources
[group('example-1')]
delete-example-1:
  kubectl delete -f ./crossplane/example-1/app-xr.yaml
  kubectl delete -f ./crossplane/example-1/app-composition.yaml
  kubectl delete -f ./crossplane/example-1/app-xrd.yaml


# Build and push the object-lister-js application
[group('example-1')]
build-object-lister tag="v1":
  cd ./object-lister-js && docker build -t {{oci_reg}}/object-lister:{{tag}} . --push

# Build and puch KCL module app_registration.
[group('example-1')]
push-app-registration-module:
  cd ./kcl-modules/app_registration && kcl mod push oci://{{oci_reg}}/kcl-app-registration

# Build and puch KCL module service_account.
[group('example-1')]
push-service-account-module:
  cd ./kcl-modules/service_account && kcl mod push oci://{{oci_reg}}/kcl-service-account

# Build and puch KCL module std_deployment.
[group('example-1')]
push-std-deployment-module:
  cd ./kcl-modules/std_deployment && kcl mod push oci://{{oci_reg}}/kcl-std-deployment

# Build and puch KCL module storage_bucket.
[group('example-1')]
push-storage-bucket-module:
  cd ./kcl-modules/storage_bucket && kcl mod push oci://{{oci_reg}}/kcl-storage-bucket

# Build and puch all KCL module.
[group('example-1')]
push-all-kcl-modules: push-app-registration-module push-service-account-module push-std-deployment-module push-storage-bucket-module
