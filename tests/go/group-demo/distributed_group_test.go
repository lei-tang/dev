package group_demo

import (
	"github.com/golang/glog"
	"gopkg.in/square/go-jose.v2"
	"testing"
)

func TestGroupToken(t *testing.T) {
	keyFile := "testdata/rsa_1.pem"
	oidcConfig := `{
	  "issuer": "{{.SERVER_URL}}",
		"jwks_uri": "{{.SERVER_URL}}/jwks"
	}`
	claims := map[string]string{"groups": "group1"}

	// load the private key for signing JWT
	privKey, err := loadJSONWebPrivateKeyFromFile(keyFile, jose.RS256)
	if err != nil {
		t.Fatalf("Failed to load private key from file: %v", err)
	}
	glog.V(5).Infof("public key is: %+v", privKey.Public())

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.SignatureAlgorithm(privKey.Algorithm),
		Key: privKey,}, nil)
	if err != nil {
		t.Fatalf("Failed to create a signer: %v", err)
	}

	pubKey := privKey.Public()
	pubKeys := []*jose.JSONWebKey{&pubKey}

	oidcServer := NewOidcTestServer(t, convertWebKeyArrayToWebKeySet(pubKeys), oidcConfig, signer,
		claims, "token", true)

	defer oidcServer.httpServer.Close()

	//authenticator, err := CreateGroupAuthenticator("Issuer-URL", "my-client-id", "groups", rootCaFilePath)
	//if err != nil {
	//	return
	//}
	//token := "jwt token"
	//userInfo, verified, err := authenticator.AuthenticateToken(token)
	//if err != nil {
	//	glog.Errorf("Failed to authenticate the token: %v", err)
	//	return
	//}
	//if !verified {
	//	glog.Errorf("The token failed to pass the verification.")
	//	return
	//}
	//glog.Errorf("The token verification succeeds.")
	//glog.Infof("The user groups is: %+v", userInfo.GetGroups())
}
