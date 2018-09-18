package main
import (
"os"
"fmt"
"time"
"math/big"
"github.com/TOSIO/go-tos/devbase/common"
"github.com/TOSIO/go-tos/devbase/crypto"
"github.com/TOSIO/go-tos/sdag/core/types"
)
func main() {
var header types.BlockHeader
var out []types.TxOut=make([]types.TxOut,2)
var testPrivHex = "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"

key, _ := crypto.HexToECDSA(testPrivHex)
tx1 :=types.TxBlock{}

//assignment begin
links :=[]common.Hash{tx1.Hash(false),tx1.Hash(false)}
out[0].Amount=big.NewInt(1)
out[1].Amount=big.NewInt(2)
tx1.Payload=[]byte("hello")

header.Type=uint32(2)
header.Time=big.NewInt(time.Now().Unix())
header.GasPrice=big.NewInt(1956)
header.GasLimit=uint64(3)
fmt.Printf("**************header:%#v\n",header)

tx1.Links=links
tx1.AccountNonce=uint64(3)
tx1.Header=header
tx1.Outs=out

//assignment end

fmt.Printf("assignment:%#v\n\n",tx1)

//signature
sign_tx,err :=tx1.Sign(key)
if err != nil {
	fmt.Println("failed\n")
	os.Exit(1)
}
fmt.Printf("**************SignTxBlock:%#v\n\n",*sign_tx)

//get hash
hash :=sign_tx.Hash(false)
fmt.Println("****hash:",hash,"\n\n")

//sig,_ :=tx1.GetSign(&tx1, key) 
sig,err :=sign_tx.GetSign(key) 
if err != nil {
  fmt.Println("failed\n")
  os.Exit(1)
}


fmt.Println("****sig:",sig,"\n\n")

pubkey,err :=tx1.GetPublicKey(hash,sig)
if err != nil {
  fmt.Println("failed\n")
  os.Exit(1)
}


fmt.Println("*********public key:",pubkey,"\n\n")
//var v bool
//v :=tx1.VerifySignature(pubkey,hash.Bytes(),sig)
sig1 :=sig[:(len(sig)-1)]
v :=tx1.VerifySignature(pubkey,hash.Bytes(),sig1)
fmt.Println("************verify result:",v,"\n\n")

diff :=tx1.Diff(hash)
fmt.Println("************hash********:",diff)



/*
var aa types.TxBlock
bb1,_:=aa.EncodeRLP(&tx)
fmt.Printf("**************:%#v\n",bb1)
err:=aa.DecodeRLP(bb1,&r)
fmt.Printf("**************:%#v\n",r,err)
sender,_:=tx1.Sender(sign)
fmt.Println("*****************Sender:",sender,"\n")
*/
/*
fmt.Println("1**********")
tx.Header1()
fmt.Println("2**********")

tx.Header2(&tx1)
fmt.Println("3**********")
a.Header3(&tx1)
*/
}
