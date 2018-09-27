package main

import (
	"encoding/json"
	"flag"
	"github.com/golang/glog"
	"github.com/lei-tang/dev/tests/go/group-demo-2/utils"
	"gopkg.in/square/go-jose.v2"
)

var (
	tokenServiceIssuer = `"token-service"`
)

func TestDistributedGroupToken(oidcServerUrl, rootCaCertPath, jwt string) {
	glog.V(5).Infof("Enter TestDistributedGroupToken")

	// Check whether the JWT contains a distributed groups claim
	// If not, no need to resolve the distributed groups claim
	containDistGroupClaim, err := utils.ContainDistributedGroupsClaim(jwt, "groups")
	if err != nil {
		glog.Fatalf("Failed on getting distributed groups claim: %v", err)
	}
	if !containDistGroupClaim {
		glog.Fatalf("The test JWT contains a distributed groups claim but the function returns false.")
	}
	glog.V(5).Infof("The JWT contains distributed groups claim.")

	// Parse the JWT issuer
	issuerUrl, err := utils.GetJwtIss(jwt)
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
	userInfo, claims, verified, err := authenticator.AuthenticateToken(jwt)
	if err != nil {
		glog.Errorf("Failed to authenticate the JWT: %v", err)
		return
	}
	if !verified {
		glog.Errorf("The JWT failed to pass the authentication.")
		return
	}
	glog.Infof("The JWT is authenticated.")
	glog.Infof("The user groups is: %+v", userInfo.GetGroups())
	glog.Infof("The claims are: %+v", claims)

	glog.Infof("Sign the resolved JWT claims.")
	// Load the private key for signing resolved JWT
	keySet, err := utils.LoadJSONWebKeySetFromJson("../testdata/token_service_signing_key_jwks.json")
	if err != nil {
		glog.Fatalf("Failed to load JSON key from the file: %v", err)
	}
	if len(keySet.Keys) < 1 {
		glog.Fatalf("Empty key set!")
	}
	if len(keySet.Keys) > 1 {
		glog.Infof("Multiple keys in the key set. Only the first one is used.")
	}
	privKey := keySet.Keys[0]
	glog.V(5).Infof("public key is: %+v", privKey.Public())
	glog.Infof("Private key is: %+v", privKey)
	// TODO: the private key must be of rsa.PrivateKey format
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.SignatureAlgorithm(jose.RS256),
		               Key: privKey.Key}, nil)
	if err != nil {
		glog.Fatalf("Failed to create a signer: %v", err)
	}
	// Replace the issuer to be token service.
	if _,ok := claims["iss"]; ok {
		glog.V(5).Infof("Replace issuer from: %+v", string(claims["iss"]))
		claims["iss"] = []byte(tokenServiceIssuer)
		glog.V(5).Infof("to: %+v", string(claims["iss"]))
	}
	// Sign the resolved JWT
	jwtByte, err := json.Marshal(claims)
	if err != nil {
		glog.Fatalf("Failed to convert claims to JSON: %v", err)
	}
	jwtResolved, err := utils.CreateTestJwt(string(jwtByte), tokenServiceIssuer, signer)
	if err != nil {
		glog.Fatalf("Failed to create a JWT: %v", err)
	}
	glog.Infof("The JWT with resolved claims is: %+v", jwtResolved)
}

//TODO:
//1. clean up the log, show the demo related log with -v 1
//2. resign the JWT with token-service key
//3. authorize the resigned JWT with group array (follow the user guide for groups claim)
//4. write script to run the demo
//5. create the detailed message flow slide
//6. currently only resolve the "groups" claim. May extend to support any distributed claim.
func main() {
	var oidcServer string
	var rootCaCertPath string
	var jwt string
	flag.StringVar(&oidcServer, "oidc-server", "", "oidc-server URL")
	flag.StringVar(&rootCaCertPath, "tls-cert-path", "", "path to the root CA certificate")
	flag.StringVar(&jwt, "jwt", "", "the JWT to authenticate")
	flag.Parse()
	if len(oidcServer) == 0 {
		glog.Fatalf("Must specify the OIDC server URL --oidc-server.")
	}
	if len(rootCaCertPath) == 0 {
		glog.Fatalf("Must specify the path to the root CA certificate --tls-cert-path.")
	}
	if len(jwt) == 0 {
		glog.Fatalf("Must specify the JWT to authenticate --jwt.")
	}
	TestDistributedGroupToken(oidcServer, rootCaCertPath, jwt)
}
