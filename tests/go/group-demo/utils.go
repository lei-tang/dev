package group_demo

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"github.com/golang/glog"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
	// The original oidc library needs to add NewAuthenticatorWithPubKey() interface.
	// The existing New(opts Options) interface will wait 10 seconds before initializing the verifier.
	"github.com/lei-tang/dev/tests/go/group-demo/oidc"
	"text/template"
)

//CreateGroupAuthenticator() creates an OIDC authenticator for a distributed group
//claim.
//issuerUrl: the issuer for the JWT token
//clientId: OIDC client id
//groupsClaim: the key for groupsClaim, e.g., "groups"
//rootCaFilePath: the file path to the root CA certificate
//pubKeys: the public key for the verifier
func CreateGroupAuthenticator(issuerUrl, clientId, groupsClaim, userNameClaim, rootCaFilePath string,
	pubKeys []*jose.JSONWebKey) (*oidc.Authenticator, error) {
	//This is needed to avoid the error of "verifier not initialized for issuer"
	oidc.SetSynchronizeTokenIDVerifier(true)
	options := oidc.Options{
		IssuerURL:     issuerUrl,
		ClientID:      clientId,
		GroupsClaim:   groupsClaim,
		UsernameClaim: userNameClaim,
		CAFile:        rootCaFilePath,
	}

	authenticator, err := oidc.NewAuthenticatorWithPubKey(options, pubKeys)
	if err != nil {
		glog.Errorf("Failed to create an oidc authenticator: %v", err)
		return nil, err
	}

	return authenticator, nil
}

// loadJSONWebPrivateKeyFromFile creates a JSONWebKey from the private key
// in the file.
// path: the path to the private key file
// alg: the signature algorithm
func loadJSONWebPrivateKeyFromFile(path string, alg jose.SignatureAlgorithm) (*jose.JSONWebKey, error) {
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

func createTestJwt(claimJson, issuerURL string, signer jose.Signer) (string, error) {
	value := struct{ ISSUER_URL string }{ISSUER_URL: issuerURL}
	s, err := replaceValueInTemplate(claimJson, &value)
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

// replaceValueInTemplate replaces a templated input value with the actual value.
func replaceValueInTemplate(input string, value interface{}) (string, error) {
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

func convertWebKeyArrayToWebKeySet(keyArray []*jose.JSONWebKey) jose.JSONWebKeySet {
	set := jose.JSONWebKeySet{}
	for _, k := range keyArray {
		set.Keys = append(set.Keys, *k)
	}
	return set
}
