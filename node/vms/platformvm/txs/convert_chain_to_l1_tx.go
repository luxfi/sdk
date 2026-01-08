// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	consensusctx "github.com/luxfi/consensus/context"

	"bytes"
	"errors"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/node/vms/components/verify"
	"github.com/luxfi/sdk/node/vms/platformvm/signer"
	"github.com/luxfi/sdk/node/vms/platformvm/warp/message"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/vms/types"
)

const MaxChainAddressLength = 4096

var (
	_ UnsignedTx                                 = (*ConvertChainToL1Tx)(nil)
	_ utils.Sortable[*ConvertChainToL1Validator] = (*ConvertChainToL1Validator)(nil)

	ErrConvertPermissionlessChain          = errors.New("cannot convert a permissionless chain")
	ErrAddressTooLong                      = errors.New("address is too long")
	ErrConvertMustIncludeValidators        = errors.New("conversion must include at least one validator")
	ErrConvertValidatorsNotSortedAndUnique = errors.New("conversion validators must be sorted and unique")
	ErrZeroWeight                          = errors.New("validator weight must be non-zero")
)

type ConvertChainToL1Tx struct {
	// Metadata, inputs and outputs
	BaseTx `serialize:"true"`
	// ID of the Chain to transform
	Chain ids.ID `serialize:"true" json:"chainID"`
	// Blockchain where the Chain manager lives
	ManagerChainID ids.ID `serialize:"true" json:"managerChainID"`
	// Address of the Chain manager
	Address types.JSONByteSlice `serialize:"true" json:"address"`
	// Initial pay-as-you-go validators for the Chain
	Validators []*ConvertChainToL1Validator `serialize:"true" json:"validators"`
	// Authorizes this conversion
	ChainAuth verify.Verifiable `serialize:"true" json:"chainAuthorization"`
}

func (tx *ConvertChainToL1Tx) SyntacticVerify(ctx *consensusctx.Context) error {
	switch {
	case tx == nil:
		return ErrNilTx
	case tx.SyntacticallyVerified:
		// already passed syntactic verification
		return nil
	case tx.Chain == constants.PrimaryNetworkID:
		return ErrConvertPermissionlessChain
	case len(tx.Address) > MaxChainAddressLength:
		return ErrAddressTooLong
	case len(tx.Validators) == 0:
		return ErrConvertMustIncludeValidators
	case !utils.IsSortedAndUnique(tx.Validators):
		return ErrConvertValidatorsNotSortedAndUnique
	}

	if err := tx.BaseTx.SyntacticVerify(ctx); err != nil {
		return err
	}
	for _, vdr := range tx.Validators {
		if err := vdr.Verify(); err != nil {
			return err
		}
	}
	if err := tx.ChainAuth.Verify(); err != nil {
		return err
	}

	tx.SyntacticallyVerified = true
	return nil
}

func (tx *ConvertChainToL1Tx) Visit(visitor Visitor) error {
	return visitor.ConvertChainToL1Tx(tx)
}

type ConvertChainToL1Validator struct {
	// NodeID of this validator
	NodeID types.JSONByteSlice `serialize:"true" json:"nodeID"`
	// Weight of this validator used when sampling
	Weight uint64 `serialize:"true" json:"weight"`
	// Initial balance for this validator
	Balance uint64 `serialize:"true" json:"balance"`
	// [Signer] is the BLS key for this validator.
	// Note: We do not enforce that the BLS key is unique across all validators.
	//       This means that validators can share a key if they so choose.
	//       However, a NodeID + Chain does uniquely map to a BLS key
	Signer signer.ProofOfPossession `serialize:"true" json:"signer"`
	// Leftover $LUX from the [Balance] will be issued to this owner once it is
	// removed from the validator set.
	RemainingBalanceOwner message.PChainOwner `serialize:"true" json:"remainingBalanceOwner"`
	// This owner has the authority to manually deactivate this validator.
	DeactivationOwner message.PChainOwner `serialize:"true" json:"deactivationOwner"`
}

func (v *ConvertChainToL1Validator) Compare(o *ConvertChainToL1Validator) int {
	return bytes.Compare(v.NodeID, o.NodeID)
}

func (v *ConvertChainToL1Validator) Verify() error {
	if v.Weight == 0 {
		return ErrZeroWeight
	}
	nodeID, err := ids.ToNodeID(v.NodeID)
	if err != nil {
		return err
	}
	if nodeID == ids.EmptyNodeID {
		return errEmptyNodeID
	}
	return verify.All(
		&v.Signer,
		&secp256k1fx.OutputOwners{
			Threshold: v.RemainingBalanceOwner.Threshold,
			Addrs:     v.RemainingBalanceOwner.Addresses,
		},
		&secp256k1fx.OutputOwners{
			Threshold: v.DeactivationOwner.Threshold,
			Addrs:     v.DeactivationOwner.Addresses,
		},
	)
}
