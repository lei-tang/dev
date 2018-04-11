package main

import (
	"fmt"
	"github.com/hashicorp/vault/api"
	"io/ioutil"
)

// Config for prototyping purpose
const (
	istioCaMountPoint   = "istio_ca"
	istioCaDescription  = "Istio CA"
	configCaKeyCertPath = "istio_ca/config/ca"
	workloadRolePath    = "istio_ca/roles/workload_role"
	signCsrPath         = "istio_ca/sign-verbatim"
)

// Config for prototyping purpose
const (
	vaultAddrForTesting = "http://127.0.0.1:8200"
	tokenForTesting     = "myroot"
	testCAKeyCertFile   = "testdata/istio_ca.pem"
	testCsrFile         = "testdata/workload-1.csr"
)

// Get the connection to a Vault server and set the token for the connection.
// vaultAddr: the address of the Vault server (e.g., "http://127.0.0.1:8200").
// token: used for authentication.
func getVaultConnection(vaultAddr string, token string) (*api.Client, error) {
	config := api.DefaultConfig()
	config.Address = vaultAddr

	client, err := api.NewClient(config)
	if err != nil {
		fmt.Errorf("NewClient() failed (error %v)", err)
		return nil, err
	}

	//Simply sets the token for future requests without actual authentication
	client.SetToken(token)
	return client, nil
}

// Mount the Vault PKI.
// caMountPoint: the mount point for CA (e.g., "istio_ca")
// caDescription: a description for CA (e.g., "Istio CA")
func mountVaultPki(client *api.Client, caMountPoint string, caDescription string) error {
	var mountInput api.MountInput
	mountInput.Description = caDescription
	mountInput.Type = "pki"
	err := client.Sys().Mount(caMountPoint, &mountInput)
	if err != nil {
		fmt.Errorf("Mount() failed (error %v)", err)
		return err
	} else {
		return nil
	}
}

// Set the workload role that issues certs with the given max-TTL and number of key bits.
// rolePath: the path to the workload role (e.g., "istio_ca/roles/workload_role")
// maxTtl:  the max life time of a workload cert (e.g., "1h")
// keyBits:  the number of bits for the key of a workload cert (e.g., 2048)
func setWorkloadRole(client *api.Client, rolePath string, maxTtl string, keyBits int) error {
	m := map[string]interface{}{
		"max_ttl":           maxTtl,
		"key_bits":          keyBits,
		"enforce_hostnames": false,
		"allow_any_name":    true,
	}

	_, err := client.Logical().Write(rolePath, m)
	if err != nil {
		fmt.Errorf("Write() failed (error %v)", err)
		return err
	} else {
		return nil
	}
}

// Set the certificate and the private key of the CA.
// caConfigPath: the path for configuring the CA (e.g., "istio_ca/config/ca")
// keyCert: the private key and the public certificate of the CA
func setCaKeyCert(client *api.Client, caConfigPath string, keyCert string) (*api.Secret, error) {
	m := map[string]interface{}{
		"pem_bundle": keyCert,
	}

	res, err := client.Logical().Write(caConfigPath, m)
	if err != nil {
		fmt.Errorf("Write() failed (error %v)", err)
		return nil, err
	} else {
		return res, nil
	}
}

// Sign a CSR and return the signed certificate.
// csrPath: the path for signing a CSR (e.g., "istio_ca/sign-verbatim")
// csr: the CSR to be signed
func signCsr(client *api.Client, csrPath string, csr string) (*api.Secret, error) {
	m := map[string]interface{}{
		"name":                "workload_role",
		"format":              "pem",
		"use_csr_common_name": true,
		"csr": csr,
	}

	res, err := client.Logical().Write(csrPath, m)
	if err != nil {
		fmt.Errorf("Write() failed (error %v)", err)
		return nil, err
	} else {
		return res, nil
	}
}

//Run a prototyping signCsr flow, includes:
//- Create a connection to Vault
//- Mount Vault PKI
//- Set CA signing key and cert
//- Set workload role for issuing certificates
//- Sign CSR and print the certificate signed
func runProtoTypeSignCsrFlow() {
	client, err := getVaultConnection(vaultAddrForTesting, tokenForTesting)
	if err != nil {
		fmt.Errorf("getVaultConnection() failed (error %v)", err)
		return
	}

	err = mountVaultPki(client, istioCaMountPoint, istioCaDescription)
	if err != nil {
		fmt.Errorf("mountVaultPki() failed (error %v)", err)
		return
	}

	keyCert, err := ioutil.ReadFile(testCAKeyCertFile)
	if err != nil {
		fmt.Errorf("ReadFile() failed (error %v)", err)
		return
	}
	_, err = setCaKeyCert(client, configCaKeyCertPath, string(keyCert[:]))
	if err != nil {
		fmt.Errorf("setCaKeyCert() failed (error %v)", err)
		return
	}

	err = setWorkloadRole(client, workloadRolePath, "1h", 2048)
	if err != nil {
		fmt.Errorf("setWorkloadRole() failed (error %v)", err)
		return
	}

	testCsr, err := ioutil.ReadFile(testCsrFile)
	if err != nil {
		fmt.Errorf("ReadFile() failed (error %v)", err)
		return
	}
	res, err := signCsr(client, signCsrPath, string(testCsr[:]))
	if err != nil {
		fmt.Errorf("signCsr() failed (error %v)", err)
	} else {
		fmt.Println("The certificate generated from CSR is :")
		//Print the certificate
		fmt.Printf("%v", res.Data["certificate"])
	}
}

// For logging,  -stderrthreshold=INFO --alsologtostderr
// https://github.com/istio/istio/blob/master/security/cmd/istio_ca/main.go#L243

//vault mount -path=istio_ca -description="Istio CA" pki
//vault write istio_ca/config/ca  pem_bundle=@./istio_ca.pem
//vault write istio_ca/roles/workload_role max_ttl=1h key_bits=2048 enforce_hostnames=false allow_any_name=true
//vault write istio_ca/sign-verbatim name=workload_role format=pem use_csr_common_name=true csr=@workload-1.csr
func main() {
	runProtoTypeSignCsrFlow()
}
