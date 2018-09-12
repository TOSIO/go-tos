package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"

	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/services/accounts/keystore"
	"github.com/pborman/uuid"
)

func main() {
	//Create private Key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		fmt.Println("GenerateKey fail")
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
	passphrase := "pass_word"
	keyjson, err := keystore.EncryptKey(key, passphrase, keystore.StandardScryptN, keystore.StandardScryptP)
	if err != nil {
		fmt.Printf("Error encrypting key: %v\n", err)
	}

	//Write to File
	keyfilepath := "keyfile.json"
	if err := ioutil.WriteFile(keyfilepath, keyjson, 0600); err != nil {
		fmt.Printf("Failed to write keyfile to %s: %v\n", keyfilepath, err)
	}

	//Decryp key with passphrase.
	newKey, err := keystore.DecryptKey(keyjson, passphrase)
	if err != nil {
		fmt.Printf("Error decrypting key: %v\n", err)
	}

	//Change password
	newPassphrase := "new_pass_word"
	newKeyjson, err := keystore.EncryptKey(newKey, newPassphrase, keystore.StandardScryptN, keystore.StandardScryptP)

	//Write to File
	if err := ioutil.WriteFile(keyfilepath, newKeyjson, 0600); err != nil {
		fmt.Printf("Failed to write keyfile to %s: %v\n", keyfilepath, err)
	}

	//print Address/Public key/Private key
	type outputInspect struct {
		Address    string
		PublicKey  string
		PrivateKey string
	}
	out := outputInspect{
		Address: newKey.Address.Hex(),
		PublicKey: hex.EncodeToString(
			crypto.FromECDSAPub(&newKey.PrivateKey.PublicKey)),
		PrivateKey: hex.EncodeToString(crypto.FromECDSA(newKey.PrivateKey)),
	}

	fmt.Println("Address:       ", out.Address)
	fmt.Println("Public key:    ", out.PublicKey)
	fmt.Println("Private key:   ", out.PrivateKey)
}
