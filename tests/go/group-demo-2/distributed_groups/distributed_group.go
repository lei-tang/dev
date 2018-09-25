package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/lei-tang/dev/tests/go/group-demo-2/utils"
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
	userInfo, verified, err := authenticator.AuthenticateToken(jwt)
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
}

//TODO:
//1. clean up the log, show the demo related log with -v 1
//2. resign the JWT with token-service key
//3. authorize the resigned JWT with group array (follow the user guide for groups claim)
//4. write script to run the demo
//5. create the detailed message flow slide
func main() {
	var oidcServer string
	var rootCaCertPath string
	var jwt string
	flag.StringVar(&oidcServer, "oidc-server", "", "oidc-server URL")
	flag.StringVar(&rootCaCertPath, "root-ca-cert-path", "", "path to the root CA certificate")
	flag.StringVar(&jwt, "jwt", "", "the JWT to authenticate")
	flag.Parse()
	if len(oidcServer) == 0 {
		glog.Fatalf("Must specify the OIDC server URL --oidc-server.")
	}
	if len(rootCaCertPath) == 0 {
		glog.Fatalf("Must specify the path to the root CA certificate --root-ca-cert-path.")
	}
	if len(jwt) == 0 {
		glog.Fatalf("Must specify the JWT to authenticate --jwt.")
	}
	TestDistributedGroupToken(oidcServer, rootCaCertPath, jwt)
}
