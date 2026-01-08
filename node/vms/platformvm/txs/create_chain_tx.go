// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"context"
	"errors"
	"unicode"

	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/sdk/node/vms/components/verify"
)

const (
	MaxNameLen    = 128
	MaxGenesisLen = units.MiB
)

var (
	_ UnsignedTx = (*CreateChainTx)(nil)

	ErrCantValidatePrimaryNetwork = errors.New("new blockchain can't be validated by primary network")

	errInvalidVMID             = errors.New("invalid VM ID")
	errFxIDsNotSortedAndUnique = errors.New("feature extensions IDs must be sorted and unique")
	errNameTooLong             = errors.New("name too long")
	errGenesisTooLong          = errors.New("genesis too long")
	errIllegalNameCharacter    = errors.New("illegal name character")
)

// CreateChainTx is an unsigned createChainTx
type CreateChainTx struct {
	// Metadata, inputs and outputs
	BaseTx `serialize:"true"`
	// ID of the Chain that validates this blockchain
	ChainID ids.ID `serialize:"true" json:"chainID"`
	// A human readable name for the blockchain; need not be unique
	BlockchainName string `serialize:"true" json:"blockchainName"`
	// ID of the VM running on the new blockchain
	VMID ids.ID `serialize:"true" json:"vmID"`
	// IDs of the feature extensions running on the new blockchain
	FxIDs []ids.ID `serialize:"true" json:"fxIDs"`
	// Byte representation of genesis state of the new blockchain
	GenesisData []byte `serialize:"true" json:"genesisData"`
	// Authorizes this blockchain to be added to this chain
	ChainAuth verify.Verifiable `serialize:"true" json:"chainAuthorization"`
}

func (tx *CreateChainTx) SyntacticVerify(ctx *consensusctx.Context) error {
	switch {
	case tx == nil:
		return ErrNilTx
	case tx.SyntacticallyVerified: // already passed syntactic verification
		return nil
	case tx.ChainID == constants.PrimaryNetworkID:
		return ErrCantValidatePrimaryNetwork
	case len(tx.BlockchainName) > MaxNameLen:
		return errNameTooLong
	case tx.VMID == ids.Empty:
		return errInvalidVMID
	case !utils.IsSortedAndUnique(tx.FxIDs):
		return errFxIDsNotSortedAndUnique
	case len(tx.GenesisData) > MaxGenesisLen:
		return errGenesisTooLong
	}

	for _, r := range tx.BlockchainName {
		if r > unicode.MaxASCII || (!unicode.IsLetter(r) && !unicode.IsNumber(r) && r != ' ') {
			return errIllegalNameCharacter
		}
	}

	if err := tx.BaseTx.SyntacticVerify(ctx); err != nil {
		return err
	}
	if err := tx.ChainAuth.Verify(); err != nil {
		return err
	}

	tx.SyntacticallyVerified = true
	return nil
}

func (tx *CreateChainTx) Visit(visitor Visitor) error {
	return visitor.CreateChainTx(tx)
}

// InitializeWithContext initializes the transaction with consensus context
func (tx *CreateChainTx) InitializeWithContext(ctx context.Context) error {
	// Initialize any context-dependent fields here
	return nil
}
