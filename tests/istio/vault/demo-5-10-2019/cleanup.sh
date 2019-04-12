#!/bin/bash

cd ~/tools/istio/istio-release-1.1-20190509-14-54

gcloud container clusters get-credentials csm-demo-vault-ca-mtls --zone us-central1-a --project endpoints-authz-test1

kubectl delete -f httpbin-injected.yaml
kubectl delete -f sleep-injected.yaml
kubectl delete serviceaccount vault-citadel-sa
kubectl delete -f istio-auth.yaml
