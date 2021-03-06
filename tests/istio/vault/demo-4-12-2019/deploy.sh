#!/bin/bash

gcloud container clusters get-credentials csm-demo-vault-ca-mtls --zone us-central1-a --project endpoints-authz-test1

kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user="$(gcloud config get-value core/account)"

pushd ~/tools/istio/
wget https://storage.googleapis.com/istio-prerelease/daily-build/release-1.1-20190404-09-16/istio-release-1.1-20190404-09-16-linux.tar.gz
tar xfz istio-release-1.1-20190404-09-16-linux.tar.gz 
cd istio-release-1.1-20190404-09-16

cat install/kubernetes/namespace.yaml > istio-auth.yaml
cat install/kubernetes/helm/istio-init/files/crd-* >> istio-auth.yaml

helm template \
    --name=istio \
    --namespace=istio-system \
    --set global.mtls.enabled=true \
    --set global.proxy.excludeIPRanges="35.233.249.249/32" \
    --values install/kubernetes/helm/istio/example-values/values-istio-example-sds-vault.yaml \
    install/kubernetes/helm/istio >> istio-auth.yaml

kubectl create -f istio-auth.yaml

