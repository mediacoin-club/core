package chain

import (
	"bytes"

	"github.com/mediacoin-pro/core/common/bin"
	"github.com/mediacoin-pro/core/crypto"
	"github.com/mediacoin-pro/core/crypto/merkle"
)

type Block struct {
	*BlockHeader
	Txs []*Transaction `json:"txs"`
}

func NewBlock(h *BlockHeader, txs []*Transaction) *Block {
	return &Block{h, txs}
}

func GenerateNewBlock(
	bc BCContext,
	txs []*Transaction,
	prv *crypto.PrivateKey,
) (block *Block, err error) {
	return GenerateNewBlockEx(bc, txs, prv, Timestamp(), 0)
}

func GenerateNewBlockEx(
	bc BCContext,
	txs []*Transaction,
	prv *crypto.PrivateKey,
	timestamp int64,
	nonce uint64,
) (block *Block, err error) {

	st := bc.State()
	validTxs := txs[:0]
	for _, tx := range txs {
		if tx == nil {
			continue // skip
		} else if _tx, err := bc.TransactionByID(tx.ID()); err != nil {
			return nil, err
		} else if _tx != nil {
			continue // skip. tx has registered
		}
		if upd, err := tx.Execute(st); err == nil {
			tx.StateUpdates = upd
			st.Apply(upd)
			validTxs = append(validTxs, tx)
		}
	}
	if len(validTxs) == 0 {
		return nil, nil
	}

	pre := bc.LastBlockHeader()

	block = &Block{&BlockHeader{
		Version:   0,
		Network:   pre.Network,
		ChainID:   pre.ChainID,
		Num:       pre.Num + 1,
		PrevHash:  pre.Hash(),
		Timestamp: timestamp,
		Nonce:     nonce,
		Miner:     prv.PublicKey(),
	}, validTxs}

	stTree := bc.StateTree()
	for _, tx := range block.Txs {
		for _, v := range tx.StateUpdates {
			if v.ChainID == block.ChainID {
				stTree.Put(v.StateKey(), v.Balance.Bytes())
			}
		}
	}
	block.TxRoot = block.txRoot()
	block.StateRoot, err = stTree.Root()
	if err != nil {
		return nil, err
	}

	chainTree := bc.ChainTree()
	err = chainTree.PutVar(block.Num, block.Hash())
	if err != nil {
		return nil, err
	}
	block.ChainRoot, err = chainTree.Root()
	if err != nil {
		return nil, err
	}

	// set signature( b.Hash + chainRoot )
	block.Sig = prv.Sign(block.sigHash())

	return
}

// Size returns block-header size + txs size
func (b *Block) Size() int64 {
	return int64(len(b.Encode()))
}

func (b *Block) CountTxs() int {
	return len(b.Txs)
}

func (b *Block) Encode() []byte {
	return bin.Encode(b.BlockHeader, b.Txs)
}

func (b *Block) Decode(data []byte) (err error) {
	return bin.Decode(data, &b.BlockHeader, &b.Txs)
}

func (b *Block) Verify(pre *BlockHeader, bcCfg *Config) error {
	// verify block header
	if err := b.BlockHeader.VerifyHeader(pre, bcCfg); err != nil {
		return err
	}
	// verify block txs
	if err := b.verifyTxs(); err != nil {
		return err
	}
	return nil
}

func (b *Block) verifyTxs() error {
	if len(b.Txs) == 0 {
		return ErrEmptyBlock
	}
	for _, tx := range b.Txs {
		// check tx-chain info
		if tx.ChainID != b.ChainID {
			return ErrInvalidChainID
		}
		if tx.Network != b.Network {
			return ErrTxInvalidNetworkID
		}
	}
	if txRoot := b.txRoot(); !bytes.Equal(b.TxRoot, txRoot) {
		return ErrInvalidTxsMerkleRoot
	}
	return nil
}

func (b *Block) txRoot() []byte {
	var hh [][]byte
	for _, it := range b.Txs {
		hh = append(hh, it.TxStHash())
	}
	return merkle.Root(hh...)
}
