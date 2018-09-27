package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/lei-tang/dev/tests/go/group-demo-2/utils"
	"gopkg.in/square/go-jose.v2"
)

var (
	tokenServiceIssuer = `"token-service"`
	testKeyId = "DHFbpoIUqrY8t2zpA2qXfCmr5VO5ZEr4RzHU_-envvQ"
)


//TODO:
//1. clean up the log, show the demo related log with -v 1
//3. authorize the resigned JWT with group array (follow the user guide for groups claim)
//4. write script to run the demo
//5. create the detailed message flow slide
//6. currently only resolve the "groups" claim. May extend to support any distributed claim.
func main() {
	var tlsCertPath string
	var jwt string
	flag.StringVar(&tlsCertPath, "tls-cert-path", "", "path to the root CA certificate")
	flag.StringVar(&jwt, "jwt", "", "the JWT to authenticate")
	flag.Parse()
	if len(tlsCertPath) == 0 {
		glog.Fatalf("Must specify the path to the root CA certificate --tls-cert-path.")
	}
	if len(jwt) == 0 {
		glog.Fatalf("Must specify the JWT to authenticate --jwt.")
	}
	// Resolve the distributed groups claim
	userInfo, claims, err := utils.ResolveDistributedGroupToken("test-client-id", "groups",
		"", "username",	tlsCertPath, jwt)
	if err != nil {
		glog.Fatalf("Failed to resolve the distributed group token: %v", err)
	}
	glog.Infof("The JWT is authenticated.")
	glog.Infof("The user groups is: %+v", userInfo.GetGroups())
	glog.V(5).Infof("The claims are: %+v", claims)

	// Load the private key for signing resolved JWT
	privKey, err := utils.LoadJSONWebPrivateKeyFromFile("../testdata/token_service_signing_key.pem", jose.RS256)
	if err != nil {
		glog.Fatalf("Failed to load signing key: %v", err)
	}
	// Key id needs to be consistent with the JWT issuer's key id
	// https://raw.githubusercontent.com/istio/istio/master/security/tools/jwt/samples/jwks.json
	privKey.KeyID = testKeyId
	glog.V(5).Infof("public key is: %+v", privKey.Public())
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.SignatureAlgorithm(jose.RS256),
		Key: privKey}, nil)
	if err != nil {
		glog.Fatalf("Failed to create a signer: %v", err)
	}
	glog.Infof("Create a new JWT with the resolved JWT claims.")
	jwtResolved, err := utils.CreateJwtWithClaims(tokenServiceIssuer, signer, claims)
	if err != nil {
		glog.Fatalf("Failed to create a JWT with the resolved claims: %v", err)
	}
	glog.Infof("The JWT with resolved claims is: %+v", jwtResolved)
}
