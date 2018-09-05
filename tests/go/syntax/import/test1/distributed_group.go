package main

import (
	"fmt"
	"k8s.io/apiserver/plugin/pkg/authenticator/token/oidc"
)

func main() {
	options := oidc.Options{
		IssuerURL:     "url",
		ClientID:      "my-client",
		UsernameClaim: "username",
		GroupsClaim:   "groups",
	}
	fmt.Println("%+v", options)
}
