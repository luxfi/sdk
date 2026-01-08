// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package txs

// NFTStaker is an interface for stakers that use NFTs for validation
type NFTStaker interface {
	Staker
	GetValidatorNFT() *ValidatorNFTInfo
}

// ValidatorNFTInfo contains NFT information for validator staking
type ValidatorNFTInfo struct {
	ContractAddress string `json:"contractAddress"`
	TokenID         uint64 `json:"tokenId"`
	CollectionName  string `json:"collectionName"`
}
