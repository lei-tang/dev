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
	"k8s.io/apiserver/plugin/pkg/authenticator/token/oidc"
	"text/template"
)


//CreateGroupAuthenticator() creates an OIDC authenticator for a distributed group
//claim.
//issuerUrl: the issuer for the JWT token
//clientId: OIDC client id
//groupsClaim: the key for groupsClaim, e.g., "groups"
//rootCaFilePath: the file path to the root CA certificate
func CreateGroupAuthenticator(issuerUrl, clientId, groupsClaim, rootCaFilePath string) (*oidc.Authenticator, error) {
	options := oidc.Options{
		IssuerURL:   issuerUrl,
		ClientID:    clientId,
		GroupsClaim: groupsClaim,
		CAFile:      rootCaFilePath,
	}

	authenticator, err := oidc.New(options)
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

func createJwt() (string, error) {
	privKey, err := loadJSONWebPrivateKeyFromFile("testdata/rsa_1.pem", jose.RS256)
	if err != nil {
		return "", err
	}

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.SignatureAlgorithm(privKey.Algorithm),
		Key: privKey,}, nil)
	if err != nil {
		glog.Errorf("Failed to create a signer: %v", err)
	}
	glog.V(5).Infof("Signer created: %+v", signer)

	glog.V(5).Infof("public key is: %+v", privKey.Public())
	return "", nil
}


// replaceValueInTemplate replaces a templated input value with the actual value.
func replaceValueInTemplate(input string, value interface{}) (string, error) {
	tpl, err := template.New("replace-templated-value").Parse(input)
	if err != nil {
		glog.Errorf("Failed to parse templated input string: %v", err)
		return "", err
	}
	buffer := bytes.NewBuffer(nil)
	err = tpl.Execute(buffer, &value)
	if err != nil {
		glog.Errorf("Failed to replace the template: %v", err)
		return "", err
	}
	glog.V(5).Infof("Replaced the templated value %v as: %v", input, buffer.String())
	return buffer.String(), nil
}

func convertWebKeyArrayToWebKeySet(keyArray []*jose.JSONWebKey) jose.JSONWebKeySet {
	set := jose.JSONWebKeySet{}
	for _, k := range keyArray {
		set.Keys = append(set.Keys, *k)
	}
	return set
}