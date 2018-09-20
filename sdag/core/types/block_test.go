package types

import (
	"fmt"
	"testing"
)

func blockInterfaceExample(b Block) error {
	rlpByte := b.GetRlp()
	fmt.Println("tx: ",b)
	fmt.Println("txRLP: ", rlpByte)

	//sender
	sender, _ := b.GetSender()
	fmt.Println("sender: ",sender)

	//hash
	hash := b.GetHash()
	fmt.Println("hash: ", hash)

	//diff
	diff := b.GetDiff()
	fmt.Println("diff", diff)

	//验证
	err := b.Validation()

	return err
}

func TestBlockUpRlp(t *testing.T) {
	tx, _ := txBlockExample()

	if uTx, err := BlockUpRlp(tx.GetRlp()); err != nil {
		t.Errorf("tx prase error")
	}else {
		fmt.Println("uTx:  ", uTx)
	}

	mb, _ := minerBlockExample()
	if uMb, err := BlockUpRlp(mb.GetRlp()); err != nil {
		t.Errorf("mb prase error: %s" , err)
	}else {
		fmt.Println("uMb:  ", uMb)
	}
}