package utils

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/golang/glog"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
	"k8s.io/apiserver/pkg/authentication/user"
	"strings"

	// The New(opts Options) interface in the original oidc library
	// will wait 10 seconds before initializing the verifier.
	oidc "github.com/lei-tang/dev/tests/go/group-demo-2/oidc_library"
	"text/template"
)

//CreateGroupAuthenticator() creates an OIDC authenticator for a distributed group
//claim.
//issuerUrl: the issuer for the JWT token
//clientId: OIDC client id
//groupsClaim: the key for groups claim, e.g., "groups",
//when it is an empty string. groups claim will not be extracted.
//groupsPrefix: the prefix added to the groups claim.
//userNameClaim: the key for user name claim, e.g., "username", "email", and etc.
//The user name claim must be present in the JWT.
//rootCaFilePath: the file path to the root CA certificate
//requiredClaims: the claim names and values that must exist in the JWT
//pubKeys: *OBSOLETE* the public key for the verifier
//func CreateGroupAuthenticator(issuerUrl, clientId, groupsClaim, userNameClaim, rootCaFilePath string,
//	pubKeys []*jose.JSONWebKey) (*oidc.Authenticator, error) {
func CreateGroupAuthenticator(issuerUrl, clientId, groupsClaim, groupsPrefix, userNameClaim,
	rootCaFilePath string, requiredClaims map[string]string) (*oidc.Authenticator, error) {
	//This is needed to avoid the error of "verifier not initialized for issuer"
	oidc.SetSynchronizeTokenIDVerifier(true)
	options := oidc.Options{
		IssuerURL:      issuerUrl,
		ClientID:       clientId,
		GroupsClaim:    groupsClaim,
		GroupsPrefix:   groupsPrefix,
		UsernameClaim:  userNameClaim,
		CAFile:         rootCaFilePath,
		RequiredClaims: requiredClaims,
	}

	authenticator, err := oidc.NewAuthenticatorWithIssuerURL(options)
	if err != nil {
		glog.Errorf("Failed to create an oidc authenticator: %v", err)
		return nil, err
	}

	return authenticator, nil
}

// LoadJSONWebPrivateKeyFromFile creates a JSONWebKey from the private key
// in the file.
// path: the path to the private key file
// alg: the signature algorithm
func LoadJSONWebPrivateKeyFromFile(path string, alg jose.SignatureAlgorithm) (*jose.JSONWebKey, error) {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		glog.Errorf("Failed to read key file: %v", err)
		return nil, err
	}
	p, _ := pem.Decode(d)
	if p == nil {
		glog.Errorf("Failed to decode the PEM file.")
		return nil, fmt.Errorf("Failed to decode the PEM file.")
	}
	priv, err := x509.ParsePKCS1PrivateKey(p.Bytes)
	if err != nil {
		glog.Errorf("Failed to parse private key: %v", err)
		return nil, err
	}
	key := &jose.JSONWebKey{Key: priv, Algorithm: string(alg)}
	hash, err := key.Thumbprint(crypto.SHA256)
	if err != nil {
		glog.Errorf("Failed to compute a SHA256 hash for the key: %v", err)
		return nil, err
	}
	key.KeyID = hex.EncodeToString(hash)
	return key, nil
}


func CreateTestJwt(claimJson, issuerURL string, signer jose.Signer) (string, error) {
	value := struct{ ISSUER_URL string }{ISSUER_URL: issuerURL}
	s, err := ReplaceValueInTemplate(claimJson, &value)
	if err != nil {
		glog.Errorf("Failed to replace the issuer URL: %v", err)
		return "", err
	}
	signed, err := signer.Sign([]byte(s))
	if err != nil {
		glog.Errorf("Failed to sign the JWT: %v", err)
		return "", err
	}
	jwt, err := signed.CompactSerialize()
	if err != nil {
		glog.Errorf("Failed to serialize the JWT: %v", err)
		return "", err
	}
	return jwt, nil
}

// ReplaceValueInTemplate replaces a templated input value with the actual value.
func ReplaceValueInTemplate(input string, value interface{}) (string, error) {
	tpl, err := template.New("replace-templated-value").Parse(input)
	if err != nil {
		glog.Errorf("Failed to parse templated input string: %v", err)
		return "", err
	}
	glog.V(5).Infof("Before the replacement: %v", input)
	buffer := bytes.NewBuffer(nil)
	err = tpl.Execute(buffer, &value)
	if err != nil {
		glog.Errorf("Failed to replace the template: %v", err)
		return "", err
	}
	glog.V(5).Infof("After the replacement: %v", buffer.String())
	return buffer.String(), nil
}

func ConvertWebKeyArrayToWebKeySet(keyArray []*jose.JSONWebKey) jose.JSONWebKeySet {
	set := jose.JSONWebKeySet{}
	for _, k := range keyArray {
		set.Keys = append(set.Keys, *k)
	}
	return set
}

// Get the iss claim from a JWT
func GetJwtIss(jwt string) (string, error) {
	// Decoded JWT payload
	var d []byte
	var err error
	s := strings.Split(jwt, ".")
	if len(s) != 3 {
		return "", fmt.Errorf("Invalid JWT with %v components", len(s))
	}
	if len(s[1]) == 0 {
		return "", fmt.Errorf("The payload of the JWT is empty")
	}
	if d, err = base64.RawURLEncoding.DecodeString(s[1]); err != nil {
		return "", fmt.Errorf("Fail to decode the JWT payload: %v", err)
	}
	issuer := struct {
		Iss string `json:"iss"`
	}{}
	// Extract iss claim from the payload
	if err = json.Unmarshal(d, &issuer); err != nil {
		return "", fmt.Errorf("Fail to parse json: %v", err)
	}
	return issuer.Iss, nil
}

// Check whether the JWT contains a distributed groups claim
func ContainDistributedGroupsClaim(jwt, groupKey string) (bool, error) {
	// The claim key for the claims (OIDC Connect Core 1.0, section 5.6.2).
	claimNamesKey := "_claim_names"
	// Decoded JWT payload
	var d []byte
	var err error
	s := strings.Split(jwt, ".")
	if len(s) != 3 {
		return false, fmt.Errorf("Invalid JWT with %v components", len(s))
	}
	if len(s[1]) == 0 {
		return false, fmt.Errorf("The payload of the JWT is empty")
	}
	if d, err = base64.RawURLEncoding.DecodeString(s[1]); err != nil {
		return false, fmt.Errorf("Fail to on to decode JWT payload: %v", err)
	}

	m := map[string]json.RawMessage{}
	if err := json.Unmarshal(d, &m); err != nil {
		return false, fmt.Errorf("Fail to unmarshal the JWT: %v", err)
	}
	if _, ok := m[claimNamesKey]; !ok {
		return false, nil
	}

	claims := map[string]json.RawMessage{}
	if err := json.Unmarshal(m[claimNamesKey], &claims); err != nil {
		return false, fmt.Errorf("Fail to unmarshal %v: %v", claimNamesKey, err)
	}
	if _, ok := claims[groupKey]; !ok {
		return false, nil
	} else {
		return true, nil
	}
}


// Create a JWT from the claims
// issuer: issuer for the JWT
// signer: the signer for the JWT
// claims: the claims in the JWT
func CreateJwtWithClaims(issuer string, signer jose.Signer, claims map[string]json.RawMessage) (string, error) {
	// Set the issuer
	if _,ok := claims["iss"]; ok {
		claims["iss"] = []byte(issuer)
	} else {
		return "", fmt.Errorf("No issuer in the claims.")
	}
	jwtByte, err := json.Marshal(claims)
	if err != nil {
		glog.Errorf("Failed to convert claims to JSON: %v", err)
		return "", nil
	}
	// Sign the resolved JWT
	signed, err := signer.Sign(jwtByte)
	if err != nil {
		glog.Errorf("Failed to sign the JWT: %v", err)
		return "", err
	}
	jwt, err := signed.CompactSerialize()
	if err != nil {
		glog.Errorf("Failed to serialize the JWT: %v", err)
		return "", err
	}

	return jwt, nil
}


// Resolve the distributed group claim in a JWT
// clientId: oidc client id
// groupClaimName: the name of the distributed group claim
// groupPrefixToAdd: the prefix to be added to a resolved distributed group claim value
// userNameClaimName: the name of the user name claim (e.g., email, username, etc)
// tlsCertPath: the path to the TLS certificate of the OIDC server
// jwt: the JWT to resolve
func ResolveDistributedGroupToken(clientId, groupClaimName, groupPrefixToAdd,
     userNameClaimName, tlsCertPath, jwt string) (user.Info, map[string]json.RawMessage, error) {
	glog.V(5).Infof("Enter ResolveDistributedGroupToken")

	// Check whether the JWT contains a distributed groups claim
	// If not, no need to resolve the distributed groups claim
	containDistGroupClaim, err := ContainDistributedGroupsClaim(jwt, "groups")
	if err != nil {
		return nil, nil, err
	}
	if !containDistGroupClaim {
		return nil, nil, fmt.Errorf("There is no distributed groups claim in the JWT")
	}

	// Parse the JWT issuer
	issuerUrl, err := GetJwtIss(jwt)
	if err != nil {
		return nil, nil, err
	}

	authenticator, err := CreateGroupAuthenticator(issuerUrl, clientId,
		groupClaimName, groupPrefixToAdd, userNameClaimName, tlsCertPath, map[string]string{})
	if err != nil {
		return nil, nil, err
	}
	glog.V(5).Infof("Authenticator has been created: %+v", authenticator)
	// Close the authenticator
	defer authenticator.Close()

	// Authenticate the group JWT token and return the resolved group info
	userInfo, claims, verified, err := authenticator.AuthenticateToken(jwt)
	if err != nil {
		return nil, nil, err
	}
	if !verified {
		return nil, nil, fmt.Errorf("The JWT failed to pass the authentication.")
	}
	return userInfo, claims, nil
}
