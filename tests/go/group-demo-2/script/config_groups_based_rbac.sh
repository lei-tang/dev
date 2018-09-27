#!/bin/bash

# Revised from istio.github.io/docs/tasks/security/rbac-groups/

# Prerequisite: install Istio and export the path to istioctl
# This script works on the Istio version: istio-master-20180920-09-15

# Configure the cluster
gcloud container clusters get-credentials distributed-group-demo1 --zone us-west1-b --project lt-istio-dev1
kubectl create ns $NS
sleep 0.3

# Enter the directory containing Istio 2018-0920 version install files
pushd ~/tools/istio/istio-master-20180920-09-15
kubectl apply -f <(istioctl kube-inject -f samples/httpbin/httpbin.yaml) -n $NS
kubectl apply -f <(istioctl kube-inject -f samples/sleep/sleep.yaml) -n $NS

# Apply an authentication policy to require both mutual TLS and JWT authentication for httpbin.
cat <<EOF | kubectl apply -n $NS -f -
apiVersion: "authentication.istio.io/v1alpha1"
kind: "Policy"
metadata:
  name: "require-mtls-jwt"
spec:
  targets:
  - name: httpbin
  peers:
  - mtls: {}
  origins:
  - jwt:
      issuer: "token-service"
      jwksUri: "https://raw.githubusercontent.com/istio/istio/master/security/tools/jwt/samples/jwks.json"
  principalBinding: USE_ORIGIN
EOF

# Enable the Istio RBAC for the namespace:
cat <<EOF | kubectl apply -n $NS -f -
apiVersion: "rbac.istio.io/v1alpha1"
kind: RbacConfig
metadata:
  name: default
spec:
  mode: 'ON_WITH_INCLUSION'
  inclusion:
    namespaces: ["rbac-groups-test-ns"]
EOF
sleep 0.5

# Once the RBAC policy takes effect, verify that Istio rejected the curl connection to the httpbin service:
# kubectl exec $(kubectl get pod -l app=sleep -n $NS -o jsonpath={.items..metadata.name}) -c sleep -n $NS -- curl http://httpbin.$NS:8000/ip -s -o /dev/null -w "%{http_code}\n" --header "Authorization: Bearer $TOKEN"

# To give read access to the httpbin service, create the httpbin-viewer service role
cat <<EOF | kubectl apply -n $NS -f -
apiVersion: "rbac.istio.io/v1alpha1"
kind: ServiceRole
metadata:
  name: httpbin-viewer
  namespace: rbac-groups-test-ns
spec:
  rules:
  - services: ["httpbin.rbac-groups-test-ns.svc.cluster.local"]
    methods: ["GET"]
EOF
sleep 0.5

# To assign the httpbin-viewer role to users in group1, create the bind-httpbin-viewer service role binding.
cat <<EOF | kubectl apply -n $NS -f -
apiVersion: "rbac.istio.io/v1alpha1"
kind: ServiceRoleBinding
metadata:
  name: bind-httpbin-viewer
  namespace: rbac-groups-test-ns
spec:
  subjects:
  - properties:
      request.auth.claims[groups]: "group1"
  roleRef:
    kind: ServiceRole
    name: "httpbin-viewer"
EOF
sleep 1
popd

