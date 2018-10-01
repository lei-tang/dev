#!/bin/bash

### Start Claims-Provider server
pushd ~/go/src/github.com/lei-tang/dev/tests/go/group-demo-2/oidc_server
go run oidc_server.go -logtostderr
# Open a new terminal and export the following environmental variables
export TLS_CERT_PATH="The TLS cert path outputed by the claims-provider server"
export JWT="The example JWT outputed by the claims-provider server"

### Config RBAC policy for httpbin service to require a JWT with valid groups claim
export NS=rbac-groups-test-ns
~/go/src/github.com/lei-tang/dev/tests/go/group-demo-2/script/config_groups_based_rbac.sh
# Without resolved groups claim, the curl command from sleep to httpbin fails
kubectl exec $(kubectl get pod -l app=sleep -n $NS -o jsonpath={.items..metadata.name}) -c sleep -n $NS -- curl http://httpbin.$NS:8000/ip -s -o /dev/null -w "%{http_code}\n"

### Resolve the distributed groups in the JWT
pushd ~/go/src/github.com/lei-tang/dev/tests/go/group-demo-2/distributed_groups
go run distributed_group.go -logtostderr --tls-cert-path ${TLS_CERT_PATH} -jwt ${JWT}
# With resolved groups claim, the curl command from sleep to httpbin succeeds
export TOKEN="The resolved JWT outputed by the previous step"
kubectl exec $(kubectl get pod -l app=sleep -n $NS -o jsonpath={.items..metadata.name}) -c sleep -n $NS -- curl http://httpbin.$NS:8000/ip -s -o /dev/null -w "%{http_code}\n" --header "Authorization: Bearer $TOKEN"


