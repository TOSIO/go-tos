package mainchain

type Mainchain struct {
}

func New() (*Mainchain, error) {
	return &Mainchain{}, nil
}

func (mc *Mainchain) GetLastTempMainBlkSlice() uint64 {
	return 0
}
