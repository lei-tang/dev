package main

import (
	"github.com/hashicorp/vault/api"
	"io/ioutil"
	"istio.io/istio/pkg/log"
	"fmt"
//	"k8s.io/kubernetes/pkg/util/mount"
)

const (
	istioCAMountPoint   = "istio_ca"
	istioCADescription   = "Istio CA"
	configCAKeyCertPath = "istio_ca/config/ca"
	workloadRolePath    = "istio_ca/roles/workload_role"
	signCsrPath         = "istio_ca/sign-verbatim"
)

// Config for testing purpose
const (
	vaultAddrForTesting = "http://127.0.0.1:8200"
	tokenForTesting     = "myroot"
	testCAKeyCertFile   = "testdata/istio_ca.pem"
	testCsrFile         = "testdata/workload-1.csr"
)

// Get the connection to a Vault server and set the token for the connection
func getVaultConnection(addr string, token string) (*api.Client, error) {
	config := api.DefaultConfig()
	config.Address = addr

	client, err := api.NewClient(config)
	if err != nil {
		log.Errorf("NewClient() failed (error %v)", err)
		return nil, err
	}

	//Simply sets the token for future requests without actual authentication
	client.SetToken(token)
	return client, nil
}

// Set the workload role with the given max-TTL and number of key bits
func setWorkloadRole(client *api.Client, maxTtl string, keyBits int) error {
	m := map[string]interface{}{
		"max_ttl":           maxTtl,
		"key_bits":          keyBits,
		"enforce_hostnames": false,
		"allow_any_name":    true,
	}

	_, err := client.Logical().Write(workloadRolePath, m)
	if err != nil {
		log.Errorf("Write() failed (error %v)", err)
		return err
	} else {
		return nil
	}
}

// Set the certificate and the private key of the CA
func setCAKeyCert(client *api.Client, keyCert string) (*api.Secret, error) {
	m := map[string]interface{}{
		"pem_bundle": keyCert,
	}

	res, err := client.Logical().Write(configCAKeyCertPath, m)
	if err != nil {
		log.Errorf("Write() failed (error %v)", err)
		return nil, err
	} else {
		return res, nil
	}
}

// Sign a CSR and return the signed certificate
func signCsr(client *api.Client, csr string) (*api.Secret, error) {
	m := map[string]interface{}{
		"name":                "workload_role",
		"format":              "pem",
		"use_csr_common_name": true,
		"csr": csr,
	}

	res, err := client.Logical().Write(signCsrPath, m)
	if err != nil {
		log.Errorf("Write() failed (error %v)", err)
		return nil, err
	} else {
		return res, nil
	}
}


// For logging,  -stderrthreshold=INFO --alsologtostderr
// https://github.com/istio/istio/blob/master/security/cmd/istio_ca/main.go#L243

//vault mount -path=istio_ca -description="Istio CA" pki
//vault write istio_ca/config/ca  pem_bundle=@./istio_ca.pem
//vault write istio_ca/roles/workload_role max_ttl=1h key_bits=2048 enforce_hostnames=false allow_any_name=true
//vault write istio_ca/sign-verbatim name=workload_role format=pem use_csr_common_name=true csr=@workload-1.csr
func main() {
	log.Debug("Start testing.")
	client, err := getVaultConnection(vaultAddrForTesting, tokenForTesting)
	if err != nil {
		log.Errorf("getVaultConnection() failed (error %v)", err)
		return
	}

	var mountInput api.MountInput
	mountInput.Description = istioCADescription
	mountInput.Type = "pki"
	// Mount the Istio PKI
	err = client.Sys().Mount(istioCAMountPoint, &mountInput)
	if err != nil {
		log.Errorf("Mount() failed (error %v)", err)
		return
	}

	// Call setCAKeyCert()
	keyCert, err := ioutil.ReadFile(testCAKeyCertFile)
	if err != nil {
		log.Errorf("ReadFile() failed (error %v)", err)
		return
	}
	_, err = setCAKeyCert(client, string(keyCert[:]))
	if err != nil {
		log.Errorf("setCAKeyCert() failed (error %v)", err)
	} else {
		log.Debug("setCAKeyCert() succeeds.")
		//log.Debugf("%#v", *res)
		//fmt.Printf("%v", res.Data["certificate"])
		//fmt.Printf("%#v", *res)
	}

	// Call setWorkloadRole()
	err = setWorkloadRole(client, "1h", 2048)
	if err != nil {
		log.Errorf("setWorkloadRole() failed (error %v)", err)
		return
	}

	// Call signCsr()
	testCsr, err := ioutil.ReadFile(testCsrFile)
	if err != nil {
		log.Errorf("ReadFile() failed (error %v)", err)
		return
	}
	res, err := signCsr(client, string(testCsr[:]))
	if err != nil {
		log.Errorf("signCsr() failed (error %v)", err)
	} else {
		log.Debug("signCsr() succeeds.")
		log.Debugf("%#v", *res)
		fmt.Printf("%v", res.Data["certificate"])
		//fmt.Printf("%#v", *res)
	}
}
