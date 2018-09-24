package main

import (
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/lei-tang/dev/tests/go/group-demo-2/utils"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"syscall"
)

var (
	testGroupResp = `{
	  "iss": "{{.ISSUER_URL}}",
	  "aud": "test-client-id",
	  "groups": ["g1", "g2"],
	  "exp": 10413792000
	}`
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
func NewOidcTestServer(pubKey jose.JSONWebKeySet, oidcConfig string, signer jose.Signer,
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
				glog.Errorf("Failed to marshal jwks: %v", err)
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
				glog.Errorf("The request token %v does not match the expected token %v", reqToken, bearerToken)
			}
			if _, ok := claims[claimName]; !ok {
				glog.Errorf("The request claim %v is invalid", claimName)
			}
			glog.V(5).Infof("claims[claimName] is %v", claims[claimName])
			signedClaim, err := signer.Sign([]byte(claims[claimName]))
			if err != nil {
				glog.Errorf("Failed to sign the claim JWT: %v", err)
			}
			jwt, err := signedClaim.CompactSerialize()
			if err != nil {
				glog.Errorf("Failed to compact-serialize the signed claim: %v", err)
			}
			resp.Write([]byte(jwt))
		default:
			resp.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(resp, "The request contains invalid URL: %v", req.URL)
		}
	}))

	glog.Infof("Serving OIDC at: %v", oidcServer.httpServer.URL)

	value := struct{ ISSUER_URL string }{ISSUER_URL: oidcServer.httpServer.URL}
	if replaceIssuerUrl {
		s, err := utils.ReplaceValueInTemplate(oidcServer.oidcConfig, &value)
		if err != nil {
			glog.Errorf("Failed to replace OIDC config: %v", err)
		}
		oidcServer.oidcConfig = s
		if _, ok := claims["groups"]; ok {
			g, err := utils.ReplaceValueInTemplate(claims["groups"], &value)
			if err != nil {
				glog.Errorf("Failed to replace groups claim: %v", err)
			}
			claims["groups"] = g
		}
	}
	return oidcServer
}

func init() {
	// Parse the flags for glog
	flag.Parse()
}

func main() {
	glog.V(5).Infof("Start OIDC server...")

	// Load the private key for signing JWT
	privKey, err := utils.LoadJSONWebPrivateKeyFromFile("../testdata/rsa_1.pem", jose.RS256)
	if err != nil {
		glog.Fatalf("Failed to load private key from file: %v", err)
	}
	glog.V(5).Infof("public key is: %+v", privKey.Public())

	// Create OIDC test server
	pubKey := privKey.Public()
	pubKeys := []*jose.JSONWebKey{&pubKey}
	oidcConfig := `{
	  "issuer": "{{.ISSUER_URL}}",
      "jwks_uri": "{{.ISSUER_URL}}/jwks"
	}`
	claims := map[string]string{"groups": testGroupResp}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.SignatureAlgorithm(privKey.Algorithm),
		Key: privKey}, nil)
	if err != nil {
		glog.Fatalf("Failed to create a signer: %v", err)
	}
	oidcServer := NewOidcTestServer(utils.ConvertWebKeyArrayToWebKeySet(pubKeys), oidcConfig, signer,
		claims, "group_access_token", true)

	// Create the CA certificate
	tempCaFile, err := ioutil.TempFile("", "temp_ca.cert")
	if err != nil {
		glog.Fatalf("Failed to create a temporary file: %v", err)
	}
	caCert := oidcServer.httpServer.TLS.Certificates[0].Certificate[0]
	pemBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert,
	}
	if err = pem.Encode(tempCaFile, pemBlock); err != nil {
		glog.Fatalf("Failed to encode the CA certificate: %v", err)
	}
	tempCaFile.Close()
	glog.Infof("The path to the root CA certificate is: %v", tempCaFile.Name())
	defer os.Remove(tempCaFile.Name())

	// Close the OIDC server when ctrl-c is pressed.
	var stopCh = make(chan os.Signal)
	signal.Notify(stopCh, syscall.SIGTERM)
	signal.Notify(stopCh, syscall.SIGINT)
	sig := <-stopCh
	glog.Infof("Caught sig: %+v", sig)
	glog.Infof("Close the server.")
	oidcServer.httpServer.Close()
	glog.Flush()
}
