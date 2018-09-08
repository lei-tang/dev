package group_demo

import (
	"encoding/pem"
	"github.com/golang/glog"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
	"os"
	"testing"
)

var (
	testClaim1 = `{
				"iss": "{{.ISSUER_URL}}",
				"aud": "test-client-id",
				"username": "test-user-name",
				"_claim_names": {
						"groups": "group1"
				},
				"_claim_sources": {
						"group1": {
								"endpoint": "{{.ISSUER_URL}}/groups",
								"access_token": "group_access_token"
						}
				},
				"exp": 10413792000
		}`

	testGroupResp = `{
					"iss": "{{.ISSUER_URL}}",
				  "aud": "test-client-id",
					"groups": ["g1", "g2"],
					"exp": 10413792000
		}`
)

func TestGroupToken(t *testing.T) {
	glog.V(5).Infof("Enter TestGroupToken")
	// Load the private key for signing JWT
	privKey, err := loadJSONWebPrivateKeyFromFile("testdata/rsa_1.pem", jose.RS256)
	if err != nil {
		t.Fatalf("Failed to load private key from file: %v", err)
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
		Key: privKey,}, nil)
	if err != nil {
		t.Fatalf("Failed to create a signer: %v", err)
	}
	oidcServer := NewOidcTestServer(t, convertWebKeyArrayToWebKeySet(pubKeys), oidcConfig, signer,
		claims, "group_access_token", true)
	defer oidcServer.httpServer.Close()

	// Create the CA certificate
	tempCaFile, err := ioutil.TempFile("", "temp_ca.cert")
	if err != nil {
		t.Fatalf("Failed to create a temporary file: %v", err)
	}
	caCert := oidcServer.httpServer.TLS.Certificates[0].Certificate[0]
	pemBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert,
	}
	if err = pem.Encode(tempCaFile, pemBlock); err != nil {
		t.Fatalf("Failed to encode the CA certificate: %v", err)
	}
	tempCaFile.Close()
	defer os.Remove(tempCaFile.Name())

	// Create the authenticator
	issuerUrl := oidcServer.httpServer.URL
	authenticator, err := CreateGroupAuthenticator(issuerUrl, "test-client-id",
		"groups", "username", tempCaFile.Name())
	if err != nil {
		t.Fatalf("Failed to create a group authenticator: %v", err)
	}
	glog.V(5).Infof("Authenticator has been created: %+v", authenticator)

	// Create a test JWT
	testJwt, err := createTestJwt(testClaim1, issuerUrl, signer)
	if err != nil {
		t.Fatalf("Failed to create a group authenticator: %v", err)
	}

	// Authenticate the group JWT token and return the resolved group info
	userInfo, verified, err := authenticator.AuthenticateToken(testJwt)
	if err != nil {
		glog.Errorf("Failed to authenticate the token: %v", err)
		return
	}
	if !verified {
		glog.Errorf("The token failed to pass the verification.")
		return
	}
	glog.Errorf("The token verification succeeds.")
	glog.Infof("The user groups is: %+v", userInfo.GetGroups())
}
