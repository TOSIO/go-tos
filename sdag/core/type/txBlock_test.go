package types
import (
	"fmt"
	"math/big" 
	"crypto/ecdsa"
	"io/ioutil"
	"os"
/*
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"io/ioutil"
	"math/big"
	"os"
*/	
	"testing"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/common"
)
var testAddrHex = "970e8128ab834e8eac17ab8e3812f010678cf791"
var testPrivHex = "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"

func TestKeyEncryptDecrypt(t *testing.T) {
fmt.Println("hello")
}
/*
func TestSign(t *testing.T) {
	key, _ := crypto.HexToECDSA(testPrivHex)
	addr := common.HexToAddress(testAddrHex)

	msg := crypto.Keccak256([]byte("foo"))
	sig, err := Sign(msg, key)
	if err != nil {
		t.Errorf("Sign error: %s", err)
	}
	recoveredPub, err := crypto.Ecrecover(msg, sig)
	if err != nil {
		t.Errorf("ECRecover error: %s", err)
	}
	pubKey := crypto.ToECDSAPub(recoveredPub)
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	if addr != recoveredAddr {
		t.Errorf("Address mismatch: want: %x have: %x", addr, recoveredAddr)
	}

	// should be equal to SigToPub
	recoveredPub2, err := crypto.SigToPub(msg, sig)
	if err != nil {
		t.Errorf("ECRecover error: %s", err)
	}
	recoveredAddr2 := crypto.PubkeyToAddress(*recoveredPub2)
	if addr != recoveredAddr2 {
		t.Errorf("Address mismatch: want: %x have: %x", addr, recoveredAddr2)
	}
}
*/
func TestPrivateKey(t *testing.T) {
		var privateKey *ecdsa.PrivateKey
		var err error
		{
			// Load private key from file.
			privateKey, err = crypto.LoadECDSA("testdata/key")
			if err != nil {
				fmt.Println("********************error")
			}
			fmt.Println(privateKey)
		}
}

func TestTxBlock(t *testing.T) {
	var  Type uint32=2
//	var prv *ecdsa.PrivateKey
//	var ss Signer
	Time :=new(big.Int).SetUint64(uint64(3))

	GasPrice :=new(big.Int).SetUint64(uint64(4))
	var GasLimit uint64=5
	Difficulty :=new(big.Int).SetUint64(uint64(6))
	r :=new(big.Int).SetUint64(uint64(6))
	s :=new(big.Int).SetUint64(uint64(6))
	v :=new(big.Int).SetUint64(uint64(6))
	tx :=TxBlock{Header:BlockHeader{Type,Time,GasPrice,GasLimit,Difficulty},Links:[]common.Hash{},AccountNonce:8,Outs:[]TxOut{},Payload:[]byte("hello"),R:r,S:s,V:v}

//	*prv=common.FromHex(testPrivHex)
// 	SignTx(&tx, ss , prv )
	
	fmt.Printf("%#v",tx)
}

func Testkey(t *testing.T) {


//	keyBytes := common.FromHex(testPrivHex)
	fileName0 := "test_key0"
//	fileName1 := "test_key1"


	ioutil.WriteFile(fileName0, []byte(testPrivHex), 0600)
	defer os.Remove(fileName0)

	key0, err := crypto.LoadECDSA(fileName0)
	if err != nil {
		fmt.Println("load key file failed")
	}
	fmt.Println(key0)
	}