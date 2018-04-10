package main

import (
	"io/ioutil"
	"log"
	"github.com/hashicorp/vault/api"
)

const (
	testCsrFile = "testdata/workload-1.csr"
)

func setWorkloadRole() {
	//pathArg := "cubbyhole/mysecret"
	pathArg := "istio_ca/roles/workload_role"

	vaultCFG := api.DefaultConfig()
	vaultCFG.Address = "http://127.0.0.1:8200"

	var err error
	vClient, err := api.NewClient(vaultCFG)
	if err != nil {
		log.Fatal(err)
	}

	vClient.SetToken("myroot")
	vault := vClient.Logical()

	m := make(map[string]interface{})
	//vault write istio_ca/roles/workload_role max_ttl=1h key_bits=2048 enforce_hostnames=false allow_any_name=true
	m["max_ttl"] = "1h"
	m["key_bits"] = 2048
	m["enforce_hostnames"] = false
	m["allow_any_name"] = true

	//_, err = vault.Write(pathArg, m)
	_, err = vault.Write(pathArg, m)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("Write succeeds.")
	}

	s, err := vault.Read(pathArg)
	if err != nil {
		log.Fatal(err)
	}
	if s == nil {
		log.Fatal("secret was nil")
	}

  log.Printf("%#v", *s)
}

func signCsr() {
		//pathArg := "cubbyhole/mysecret"
	pathArg := "istio_ca/sign-verbatim"

	vaultCFG := api.DefaultConfig()
	vaultCFG.Address = "http://127.0.0.1:8200"

	var err error
	vClient, err := api.NewClient(vaultCFG)
	if err != nil {
		log.Fatal(err)
	}

	vClient.SetToken("myroot")
	vault := vClient.Logical()

	testCsr, err := ioutil.ReadFile(testCsrFile)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("ReadFile() succeeds.")
		log.Printf("%s", testCsr)
	}

//  secret := make(map[string]interface{})
//	secret["value"] = "test secret"
	m := make(map[string]interface{})
	m["name"] = "workload_role"
	m["format"] = "pem"
	m["use_csr_common_name"] = true
	m["csr"] = string(testCsr[:])

	//_, err = vault.Write(pathArg, m)
	res, err := vault.Write(pathArg, m)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("Write succeeds.")
		log.Printf("%#v", *res)
	}

	//s, err := vault.Read(pathArg)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//if s == nil {
	//	log.Fatal("secret was nil")
	//}
	//
	//log.Printf("%#v", *s)
}

func main() {
	setWorkloadRole()
	//signCsr()
}
