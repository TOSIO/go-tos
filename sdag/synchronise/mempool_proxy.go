package synchronise

import (
	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/TOSIO/go-tos/sdag/manager"
)

type MemPoolProxy struct {
}

func NewMempol() (*MemPoolProxy, error) {
	return &MemPoolProxy{}, nil
}

func (m *MemPoolProxy) AddBlock(data []byte) error {
	b, err := types.BlockUnRlp(data)
	if err != nil {
		return err
	}
	return manager.SyncAddBlock(b)
}
