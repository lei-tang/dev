#!/bin/bash

cd ~/tools/istio/istio-release-1.1-20190404-09-16

kubectl delete -f httpbin-injected.yaml
kubectl delete -f sleep-injected.yaml
kubectl delete serviceaccount vault-citadel-sa
kubectl delete -f istio-auth.yaml
