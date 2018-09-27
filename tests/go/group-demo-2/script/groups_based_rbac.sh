#!/bin/bash

# Run istio.github.io/docs/tasks/security/rbac-groups/
export NS=rbac-groups-test-ns
# Enter the directory containing latest Istio install files
pushd ~/tools/istio/istio-master-20180920-09-15
# The deletion can take a long time
# kubectl delete namespace $NS

kubectl create ns $NS
kubectl apply -f <(istioctl kube-inject -f samples/httpbin/httpbin.yaml) -n $NS
kubectl apply -f <(istioctl kube-inject -f samples/sleep/sleep.yaml) -n $NS

# To verify that httpbin and sleep services are running and sleep is able to reach httpbin, run the following curl command:
kubectl exec $(kubectl get pod -l app=sleep -n $NS -o jsonpath={.items..metadata.name}) -c sleep -n $NS -- curl http://httpbin.$NS:8000/ip -s -o /dev/null -w "%{http_code}\n"

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

# Connect to the httpbin service. When a valid JWT is attached, it returns the HTTP code 200.
kubectl exec $(kubectl get pod -l app=sleep -n $NS -o jsonpath={.items..metadata.name}) -c sleep -n $NS -- curl http://httpbin.$NS:8000/ip -s -o /dev/null -w "%{http_code}\n" --header "Authorization: Bearer $TOKEN"

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

# Once the RBAC policy takes effect, verify that Istio rejected the curl connection to the httpbin service:
kubectl exec $(kubectl get pod -l app=sleep -n $NS -o jsonpath={.items..metadata.name}) -c sleep -n $NS -- curl http://httpbin.$NS:8000/ip -s -o /dev/null -w "%{http_code}\n" --header "Authorization: Bearer $TOKEN"

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

# After the RBAC policy takes effect, verify the connection to the httpbin service succeeds:
kubectl exec $(kubectl get pod -l app=sleep -n $NS -o jsonpath={.items..metadata.name}) -c sleep -n $NS -- curl http://httpbin.$NS:8000/ip -s -o /dev/null -w "%{http_code}\n" --header "Authorization: Bearer $TOKEN"