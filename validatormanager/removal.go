// Copyright (C) 2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package validatormanager

import (
	"context"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	sdkwarp "github.com/luxfi/sdk/warp"

	"github.com/luxfi/evm/warp/messages"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/sdk/application"
	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/sdk/evm"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/ux"
	"github.com/luxfi/sdk/validator"
	"github.com/luxfi/warp"
	warpPayload "github.com/luxfi/warp/payload"

	"github.com/luxfi/crypto"
)

func InitializeValidatorRemoval(
	rpcURL string,
	managerAddress crypto.Address,
	generateRawTxOnly bool,
	managerOwnerAddress crypto.Address,
	privateKey string,
	validationID ids.ID,
	isPoS bool,
	uptimeProofSignedMessage *warp.Message,
	force bool,
	useACP99 bool,
) (*types.Transaction, *types.Receipt, error) {
	if isPoS {
		if useACP99 {
			if force {
				return contract.TxToMethod(
					rpcURL,
					false,
					crypto.Address{},
					privateKey,
					managerAddress,
					big.NewInt(0),
					"force POS validator removal",
					ErrorSignatureToError,
					"forceInitiateValidatorRemoval(bytes32,bool,uint32)",
					validationID,
					false, // no uptime proof if force
					uint32(0),
				)
			}
			// remove PoS validator with uptime proof
			warpMsg := uptimeProofSignedMessage
			return contract.TxToMethodWithWarpMessage(
				rpcURL,
				false,
				crypto.Address{},
				privateKey,
				managerAddress,
				warpMsg,
				big.NewInt(0),
				"POS validator removal with uptime proof",
				ErrorSignatureToError,
				"initiateValidatorRemoval(bytes32,bool,uint32)",
				validationID,
				true, // submit uptime proof
				uint32(0),
			)
		}
		if force {
			return contract.TxToMethod(
				rpcURL,
				false,
				crypto.Address{},
				privateKey,
				managerAddress,
				big.NewInt(0),
				"force POS validator removal",
				ErrorSignatureToError,
				"forceInitializeEndValidation(bytes32,bool,uint32)",
				validationID,
				false, // no uptime proof if force
				uint32(0),
			)
		}
		// remove PoS validator with uptime proof
		warpMsg := uptimeProofSignedMessage
		return contract.TxToMethodWithWarpMessage(
			rpcURL,
			false,
			crypto.Address{},
			privateKey,
			managerAddress,
			warpMsg,
			big.NewInt(0),
			"POS validator removal with uptime proof",
			ErrorSignatureToError,
			"initializeEndValidation(bytes32,bool,uint32)",
			validationID,
			true, // submit uptime proof
			uint32(0),
		)
	}
	// PoA case
	if useACP99 {
		return contract.TxToMethod(
			rpcURL,
			generateRawTxOnly,
			managerOwnerAddress,
			privateKey,
			managerAddress,
			big.NewInt(0),
			"POA validator removal initialization",
			ErrorSignatureToError,
			"initiateValidatorRemoval(bytes32)",
			validationID,
		)
	}
	return contract.TxToMethod(
		rpcURL,
		generateRawTxOnly,
		managerOwnerAddress,
		privateKey,
		managerAddress,
		big.NewInt(0),
		"POA validator removal initialization",
		ErrorSignatureToError,
		"initializeEndValidation(bytes32)",
		validationID,
	)
}

func GetUptimeProofMessage(
	network models.Network,
	aggregatorLogger luxlog.Logger,
	aggregatorQuorumPercentage uint64,
	chainID ids.ID,
	blockchainID ids.ID,
	validationID ids.ID,
	uptime uint64,
	signatureAggregatorEndpoint string,
) (*warp.Envelope, error) {
	uptimePayload, err := messages.NewValidatorUptime(validationID, uptime)
	if err != nil {
		return nil, err
	}
	addressedCall, err := warpPayload.NewAddressedCall(nil, uptimePayload.Bytes())
	if err != nil {
		return nil, err
	}
	uptimeProofUnsignedMessage, err := warp.NewMessage(
		network.ID(),
		blockchainID,
		addressedCall.Bytes(),
	)
	if err != nil {
		return nil, err
	}

	messageHexStr := hex.EncodeToString(uptimeProofUnsignedMessage.Bytes())
	return sdkwarp.SignMessage(aggregatorLogger, signatureAggregatorEndpoint, messageHexStr, "", chainID.String(), aggregatorQuorumPercentage)
}

func InitValidatorRemoval(
	ctx context.Context,
	app *application.Lux,
	network models.Network,
	rpcURL string,
	chainSpec contract.ChainSpec,
	generateRawTxOnly bool,
	ownerAddressStr string,
	ownerPrivateKey string,
	nodeID ids.NodeID,
	aggregatorLogger luxlog.Logger,
	isPoS bool,
	uptimeSec uint64,
	force bool,
	validatorManagerAddressStr string,
	useACP99 bool,
	initiateTxHash string,
	signatureAggregatorEndpoint string,
) (*warp.Message, ids.ID, *types.Transaction, error) {
	chainID, err := contract.GetNetworkID(
		app,
		network,
		chainSpec,
	)
	if err != nil {
		return nil, ids.Empty, nil, err
	}
	blockchainID, err := contract.GetBlockchainID(
		app,
		network,
		chainSpec,
	)
	if err != nil {
		return nil, ids.Empty, nil, err
	}
	managerAddress := crypto.HexToAddress(validatorManagerAddressStr)
	ownerAddress := crypto.HexToAddress(ownerAddressStr)
	validationID, err := validator.GetValidationID(
		rpcURL,
		managerAddress,
		nodeID,
	)
	if err != nil {
		return nil, ids.Empty, nil, err
	}
	if validationID == ids.Empty {
		return nil, ids.Empty, nil, fmt.Errorf("node %s is not a L1 validator", nodeID)
	}

	var unsignedMessage *warp.Message
	if initiateTxHash != "" {
		unsignedMessage, err = GetL1ValidatorWeightMessageFromTx(
			rpcURL,
			validationID,
			0,
			initiateTxHash,
		)
		if err != nil {
			return nil, ids.Empty, nil, err
		}
	}

	var receipt *types.Receipt
	if unsignedMessage == nil {
		signedUptimeProof := &warp.Envelope{}
		if isPoS {
			if uptimeSec == 0 {
				uptimeSec, err = GetL1ValidatorUptimeSeconds(rpcURL, nodeID)
				if err != nil {
					return nil, ids.Empty, nil, evm.TransactionError(nil, err, "failure getting uptime data for nodeID: %s via %s ", nodeID, rpcURL)
				}
			}
			ux.Logger.PrintToUser("Using uptime: %ds", uptimeSec)
			signedUptimeProof, err = GetUptimeProofMessage(
				network,
				aggregatorLogger,
				0,
				chainID,
				blockchainID,
				validationID,
				uptimeSec,
				signatureAggregatorEndpoint,
			)
			if err != nil {
				return nil, ids.Empty, nil, evm.TransactionError(nil, err, "failure getting uptime proof")
			}
		}
		var tx *types.Transaction
		tx, receipt, err = InitializeValidatorRemoval(
			rpcURL,
			managerAddress,
			generateRawTxOnly,
			ownerAddress,
			ownerPrivateKey,
			validationID,
			isPoS,
			&signedUptimeProof.Message, // is empty for non-PoS
			force,
			useACP99,
		)
		switch {
		case err != nil:
			if !errors.Is(err, ErrInvalidValidatorStatus) {
				return nil, ids.Empty, nil, evm.TransactionError(tx, err, "failure initializing validator removal")
			}
			ux.Logger.PrintToUser("%s", luxlog.LightBlue.Wrap("The validator removal process was already initialized. Proceeding to the next step"))
		case generateRawTxOnly:
			return nil, ids.Empty, tx, nil
		default:
			ux.Logger.PrintToUser("Validator removal initialized. InitiateTxHash: %s", tx.Hash())
		}
	} else {
		ux.Logger.PrintToUser("%s", luxlog.LightBlue.Wrap("The validator removal process was already initialized. Proceeding to the next step"))
	}

	if receipt != nil {
		unsignedMessage, err = evm.ExtractWarpMessageFromReceipt(receipt)
		if err != nil {
			return nil, ids.Empty, nil, err
		}
	}

	var nonce uint64
	if unsignedMessage == nil {
		nonce, err = GetValidatorNonce(ctx, rpcURL, validationID)
		if err != nil {
			return nil, ids.Empty, nil, err
		}
	}

	// Convert node warp message back to standalone for GetL1ValidatorWeightMessage
	var standaloneUnsignedMsg *warp.Message
	if unsignedMessage != nil {
		standaloneUnsignedMsg, err = warp.NewMessage(
			unsignedMessage.NetworkID,
			unsignedMessage.SourceChainID,
			unsignedMessage.Payload,
		)
		if err != nil {
			return nil, ids.Empty, nil, err
		}
	}

	signedMsg, err := GetL1ValidatorWeightMessage(
		network,
		aggregatorLogger,
		standaloneUnsignedMsg,
		chainID,
		blockchainID,
		managerAddress,
		validationID,
		nonce,
		0,
		signatureAggregatorEndpoint,
	)
	return &signedMsg.Message, validationID, nil, err
}

func CompleteValidatorRemoval(
	rpcURL string,
	managerAddress crypto.Address,
	generateRawTxOnly bool,
	ownerAddress crypto.Address,
	privateKey string, // not need to be owner atm
	chainValidatorRegistrationSignedMessage *warp.Message,
	useACP99 bool,
) (*types.Transaction, *types.Receipt, error) {
	if useACP99 {
		return contract.TxToMethodWithWarpMessage(
			rpcURL,
			generateRawTxOnly,
			ownerAddress,
			privateKey,
			managerAddress,
			chainValidatorRegistrationSignedMessage,
			big.NewInt(0),
			"complete validator removal",
			ErrorSignatureToError,
			"completeValidatorRemoval(uint32)",
			uint32(0),
		)
	}
	return contract.TxToMethodWithWarpMessage(
		rpcURL,
		generateRawTxOnly,
		ownerAddress,
		privateKey,
		managerAddress,
		chainValidatorRegistrationSignedMessage,
		big.NewInt(0),
		"complete validator removal",
		ErrorSignatureToError,
		"completeEndValidation(uint32)",
		uint32(0),
	)
}

func FinishValidatorRemoval(
	ctx context.Context,
	app *application.Lux,
	network models.Network,
	rpcURL string,
	chainSpec contract.ChainSpec,
	generateRawTxOnly bool,
	ownerAddressStr string,
	privateKey string,
	validationID ids.ID,
	aggregatorLogger luxlog.Logger,
	validatorManagerAddressStr string,
	useACP99 bool,
	signatureAggregatorEndpoint string,
) (*types.Transaction, error) {
	managerAddress := crypto.HexToAddress(validatorManagerAddressStr)
	chainID, err := contract.GetNetworkID(
		app,
		network,
		chainSpec,
	)
	if err != nil {
		return nil, err
	}
	signedMessage, err := GetPChainL1ValidatorRegistrationMessage(
		ctx,
		network,
		rpcURL,
		aggregatorLogger,
		0,
		chainID,
		validationID,
		false,
		signatureAggregatorEndpoint,
	)
	if err != nil {
		return nil, err
	}
	if privateKey != "" {
		if client, err := evm.GetClient(rpcURL); err != nil {
			ux.Logger.RedXToUser("failure connecting to L1 to setup proposer VM: %s", err)
		} else {
			if err := client.SetupProposerVM(privateKey); err != nil {
				ux.Logger.RedXToUser("failure setting proposer VM on L1: %v", err)
			}
			client.Close()
		}
	}
	ownerAddress := crypto.HexToAddress(ownerAddressStr)
	tx, _, err := CompleteValidatorRemoval(
		rpcURL,
		managerAddress,
		generateRawTxOnly,
		ownerAddress,
		privateKey,
		signedMessage,
		useACP99,
	)
	if err != nil {
		return nil, evm.TransactionError(tx, err, "failure completing validator removal")
	}
	if generateRawTxOnly {
		return tx, nil
	}
	return nil, nil
}
