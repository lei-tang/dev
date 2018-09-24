package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/lei-tang/dev/tests/go/group-demo-2/utils"
	"gopkg.in/square/go-jose.v2"
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
)

func TestDistributedGroupToken(oidcServerUrl, rootCaCertPath string) {
	glog.V(5).Infof("Enter TestDistributedGroupToken")

	// Load the private key for signing JWT
	privKey, err := utils.LoadJSONWebPrivateKeyFromFile("../testdata/rsa_1.pem", jose.RS256)
	if err != nil {
		glog.Fatalf("Failed to load private key from file: %v", err)
	}
	glog.V(5).Infof("public key is: %+v", privKey.Public())
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.SignatureAlgorithm(privKey.Algorithm),
		Key: privKey}, nil)
	if err != nil {
		glog.Fatalf("Failed to create a signer: %v", err)
	}

	// Create a test JWT
	testJwt, err := utils.CreateTestJwt(testClaim1, oidcServerUrl, signer)
	if err != nil {
		glog.Fatalf("Failed to create a group authenticator: %v", err)
	}

	// Check whether the JWT contains a distributed groups claim
	// If not, no need to resolve the distributed groups claim
	containDistGroupClaim, err := utils.ContainDistributedGroupsClaim(testJwt, "groups")
	if err != nil {
		glog.Fatalf("Failed on getting distributed groups claim: %v", err)
	}
	if !containDistGroupClaim {
		glog.Fatalf("The test JWT contains a distributed groups claim but the function returns false.")
	}
	glog.Infof("The JWT contains distributed groups claim.")

	// Parse the JWT issuer from the JWT receivegetJwtIssd
	issuerUrl, err := utils.GetJwtIss(testJwt)
	if err != nil {
		glog.Fatalf("Failed to parse the issuer of the JWT: %v", err)
	}

	// Create the authenticator
	// TODO: create a test case when the client id (e.g., test-client-id-2) does not match
	// the audience in the JWT. In this case, the JWT should be rejected.
	// https://openid.net/specs/openid-connect-core-1_0.html#CodeIDToken
	//authenticator, err := CreateGroupAuthenticator(issuerUrl, "test-client-id-2",
	//	"groups", "username", tempCaFile.Name())
	authenticator, err := utils.CreateGroupAuthenticator(issuerUrl, "test-client-id",
		"groups", "", "username", rootCaCertPath, map[string]string{})
	if err != nil {
		glog.Fatalf("Failed to create a group authenticator: %v", err)
	}
	glog.V(5).Infof("Authenticator has been created: %+v", authenticator)
	// Close the authenticator
	defer authenticator.Close()

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

func main() {
	var oidcServer string
	var rootCaCertPath string
	flag.StringVar(&oidcServer, "oidc-server", "", "oidc-server URL")
	flag.StringVar(&rootCaCertPath, "root-ca-cert-path", "", "path to the root CA certificate")
	flag.Parse()
	if len(oidcServer) == 0 {
		glog.Fatalf("Must specify the OIDC server URL.")
	}
	if len(rootCaCertPath) == 0 {
		glog.Fatalf("Must specify the path to the root CA certificate.")
	}
	TestDistributedGroupToken(oidcServer, rootCaCertPath)
}
