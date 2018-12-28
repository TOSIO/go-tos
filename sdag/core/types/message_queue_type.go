package types

//import "math/big"

type MQBlockInfo struct {
	BlockHash      string
	TransferDate   string
	Amount         string
	GasLimit       string
	GasPrice       string
	SenderAddr     string
	ReceiverAddr   string
	IsMiner        string
	Difficulty     string
	CumulativeDiff string
}

type MQBlockStatus struct {
	BlockHash       string
	BlockHigh       string
	IsMain          string
	ConfirmStatus   string //accept/ reject / pending
	ConfirmDate     string
	GasUsed         string
	ConfirmedHash   string
	ConfirmedHigh   string
	ConfirmedOrder  string
	ContractAddress string
}
