#!/bin/bash

# watch kubectl get pod -n istio-system

# To verify mTLS is enforced, sleep curl httpbin without mTLS should fail.
kubectl exec -it $(kubectl get pod -l app=sleep -o jsonpath='{.items[0].metadata.name}') -c istio-proxy -- curl -s -o /dev/null -w "%{http_code}" httpbin:8000/headers

# sleep curl httpbin with mTLS using the certificates from Vault CA should succeed.
kubectl exec -it $(kubectl get pod -l app=sleep -o jsonpath='{.items[0].metadata.name}') -c sleep -- curl -s -o /dev/null -w "%{http_code}" httpbin:8000/headers

