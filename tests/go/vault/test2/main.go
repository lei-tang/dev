package main

import (
	"github.com/hashicorp/vault/api"
	"io/ioutil"
	"istio.io/istio/pkg/log"
)

const (
	vaultAddrForTesting = "http://127.0.0.1:8200"
	tokenForTesting     = "myroot"
	testCsrFile         = "testdata/workload-1.csr"
	workloadRolePath    = "istio_ca/roles/workload_role"
	signCsrPath         = "istio_ca/sign-verbatim"
)

// Get the connection to a Vault server and set the token for the connection
func getVaultConnection(addr string, token string) (*api.Logical, error) {
	config := api.DefaultConfig()
	config.Address = addr

	client, err := api.NewClient(config)
	if err != nil {
		log.Errorf("NewClient() failed (error %v)", err)
		return nil, err
	}

	//Simply sets the token for future requests without actual authentication
	client.SetToken(token)
	return client.Logical(), nil
}

// Set the workload role with the given max-TTL and number of key bits
func setWorkloadRole(conn *api.Logical, maxTtl string, keyBits int) error {
	m := map[string]interface{}{
		"max_ttl":           maxTtl,
		"key_bits":          keyBits,
		"enforce_hostnames": false,
		"allow_any_name":    true,
	}

	_, err := conn.Write(workloadRolePath, m)
	if err != nil {
		log.Errorf("Write() failed (error %v)", err)
		return err
	} else {
		return nil
	}
}

// Sign a CSR and return the signed certificate
func signCsr(conn *api.Logical, csr string) (*api.Secret, error) {
	m := map[string]interface{}{
		"name":                "workload_role",
		"format":              "pem",
		"use_csr_common_name": true,
		"csr": csr,
	}

	res, err := conn.Write(signCsrPath, m)
	if err != nil {
		log.Errorf("Write() failed (error %v)", err)
		return nil, err
	} else {
		return res, nil
	}
}


// For logging,  -stderrthreshold=INFO --alsologtostderr
// https://github.com/istio/istio/blob/master/security/cmd/istio_ca/main.go#L243
func main() {
	log.Debug("Start testing.")
	conn, err := getVaultConnection(vaultAddrForTesting, tokenForTesting)
	if err != nil {
		log.Errorf("getVaultConnection() failed (error %v)", err)
		return
	}

	setWorkloadRole(conn, "1h", 2048)

	testCsr, err := ioutil.ReadFile(testCsrFile)
	if err != nil {
		log.Errorf("ReadFile() failed (error %v)", err)
		return
	}
	res, err := signCsr(conn, string(testCsr[:]))
	if err != nil {
		log.Errorf("signCsr() failed (error %v)", err)
	} else {
		log.Debug("signCsr() succeeds.")
		log.Debugf("%#v", *res)
	}
}

