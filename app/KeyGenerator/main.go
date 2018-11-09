package main

import (
	"fmt"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/node"
	"github.com/TOSIO/go-tos/services/accounts/keystore"
	"github.com/pborman/uuid"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	count      = 10
	passphrase = "12345"
)

func main() {
	fmt.Println("Enter the number (Default 10) of generated keys")
	fmt.Scanf("%d", &count)
	fmt.Printf("The number of keys that need to be generated is %d\n", count)
	path := filepath.Join(node.DefaultDataDir(), "keystore")
	err := os.MkdirAll(path, 0777)
	if err != nil {
		fmt.Println("MkdirAll fail", err)
	}

	for i := 0; i < count; i++ {
		//Create private Key
		privateKey, err := crypto.GenerateKey()
		if err != nil {
			fmt.Println("GenerateKey fail", err)
			return
		}

		// Create the keyfile object with a random UUID.
		id := uuid.NewRandom()
		key := &keystore.Key{
			Id:         id,
			Address:    crypto.PubkeyToAddress(privateKey.PublicKey),
			PrivateKey: privateKey,
		}

		// Encrypt key with passphrase.
		keyjson, err := keystore.EncryptKey(key, passphrase, keystore.StandardScryptN, keystore.StandardScryptP)
		if err != nil {
			fmt.Printf("Error encrypting key: %v\n", err)
		}

		//Write to File
		keyFile := fmt.Sprintf("keyFile%05d.json", i)
		keyFilePath := filepath.Join(path, keyFile)
		if err := ioutil.WriteFile(keyFilePath, keyjson, 0600); err != nil {
			fmt.Printf("Failed to write keyfile to %s: %v\n", keyFilePath, err)
		}
		fmt.Printf("The %dth key generation is completed\n", i)
	}
	fmt.Printf("Generated\n")
	fmt.Printf("Enter any number to exit\n")
	fmt.Scan(&count)
}
