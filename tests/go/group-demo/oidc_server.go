package group_demo

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"gopkg.in/square/go-jose.v2"
	"net/http"
	"net/http/httptest"
	"testing"
)

type OidcTestServer struct {
	oidcConfig string
	httpServer *httptest.Server
}

// NewOidcTestServer creates an OIDC server for testing purpose.
// pubKey: jwks for the server
// signer: the signing key
// claims: a map with key=claim-name and value=claim-response
// token: required access token
// replaceIssuerUrl: whether replace the templated issuer url
func NewOidcTestServer(t *testing.T, pubKey jose.JSONWebKeySet, oidcConfig string, signer jose.Signer,
	claims map[string]string, token string, replaceIssuerUrl bool) *OidcTestServer {
	oidcServer := &OidcTestServer{
		oidcConfig: oidcConfig,
	}
	oidcServer.httpServer = httptest.NewTLSServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		glog.V(5).Infof("request: %+v", *req)
		switch req.URL.Path {
		case "/.well-known/openid-configuration":
			glog.V(5).Infof("%v: returning: %+v", req.URL, oidcServer.oidcConfig)
			resp.Header().Set("Content-Type", "application/json")
			resp.Write([]byte(oidcServer.oidcConfig))
		case "/jwks":
			resp.Header().Set("Content-Type", "application/json")
			pubKeyBytes, err := json.Marshal(pubKey)
			if err != nil {
				t.Errorf("Failed to marshal jwks: %v", err)
			}
			glog.V(5).Infof("%v: returning: %+v", req.URL, string(pubKeyBytes))
			resp.Write(pubKeyBytes)
		case "/groups": //only support groups claim
			claimName := "groups"
			glog.V(5).Infof("claim name is %v", claimName)

			bearerToken := fmt.Sprintf("Bearer %v", token)
			glog.V(5).Infof("bearerToken is %v", bearerToken)

			reqToken := req.Header.Get("Authorization")
			glog.V(5).Infof("Request token is %v", reqToken)
			if bearerToken != reqToken {
				t.Errorf("The request token %v does not match the expected token %v", reqToken, bearerToken)
			}
			if _, ok := claims[claimName]; !ok {
				t.Errorf("The request claim %v is invalid", claimName)
			}
			glog.V(5).Infof("claims[claimName] is %v", claims[claimName])
			signedClaim, err := signer.Sign([]byte(claims[claimName]))
			if err != nil {
				t.Errorf("Failed to sign the claim JWT: %v", err)
			}
			jwt, err := signedClaim.CompactSerialize()
			if err != nil {
				t.Errorf("Failed to compact-serialize the signed claim: %v", err)
			}
			resp.Write([]byte(jwt))
		default:
			resp.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(resp, "The request contains invalid URL: %v", req.URL)
		}
	}))
	glog.V(4).Infof("Serving OIDC at: %v", oidcServer.httpServer.URL)

	value := struct{ ISSUER_URL string }{ISSUER_URL: oidcServer.httpServer.URL}
	if replaceIssuerUrl {
		s, err := replaceValueInTemplate(oidcServer.oidcConfig, &value)
		if err != nil {
			t.Errorf("Failed to replace OIDC config: %v", err)
		}
		oidcServer.oidcConfig = s
		if _, ok := claims["groups"]; ok {
			g, err := replaceValueInTemplate(claims["groups"], &value)
			if err != nil {
				t.Errorf("Failed to replace groups claim: %v", err)
			}
			claims["groups"] = g
		}
	}
	return oidcServer
}
