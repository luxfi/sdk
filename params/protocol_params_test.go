// (c) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package params_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/luxfi/evm/params"
	"github.com/stretchr/testify/assert"
)

// TestUpstreamParamsValues detects when a params value changes upstream to prevent a subtle change
// to one of the values to have an unpredicted impact in the libevm consumer.
// Values should be updated to newer upstream values once the consumer is updated to handle the
// updated value(s).
func TestUpstreamParamsValues(t *testing.T) {
	tests := map[string]struct {
		param any
		want  any
	}{
		"GasLimitBoundDivisor":               {param: params.GasLimitBoundDivisor, want: uint64(1024)},
		"MinGasLimit":                        {param: params.MinGasLimit, want: uint64(5000)},
		"MaxGasLimit":                        {param: params.MaxGasLimit, want: uint64(0x7fffffffffffffff)},
		"GenesisGasLimit":                    {param: params.GenesisGasLimit, want: uint64(4712388)},
		"ExpByteGas":                         {param: params.ExpByteGas, want: uint64(10)},
		"SloadGas":                           {param: params.SloadGas, want: uint64(50)},
		"CallValueTransferGas":               {param: params.CallValueTransferGas, want: uint64(9000)},
		"CallNewAccountGas":                  {param: params.CallNewAccountGas, want: uint64(25000)},
		"TxGas":                              {param: params.TxGas, want: uint64(21000)},
		"TxGasContractCreation":              {param: params.TxGasContractCreation, want: uint64(53000)},
		"TxDataZeroGas":                      {param: params.TxDataZeroGas, want: uint64(4)},
		"QuadCoeffDiv":                       {param: params.QuadCoeffDiv, want: uint64(512)},
		"LogDataGas":                         {param: params.LogDataGas, want: uint64(8)},
		"CallStipend":                        {param: params.CallStipend, want: uint64(2300)},
		"Keccak256Gas":                       {param: params.Keccak256Gas, want: uint64(30)},
		"Keccak256WordGas":                   {param: params.Keccak256WordGas, want: uint64(6)},
		"InitCodeWordGas":                    {param: params.InitCodeWordGas, want: uint64(2)},
		"SstoreSetGas":                       {param: params.SstoreSetGas, want: uint64(20000)},
		"SstoreResetGas":                     {param: params.SstoreResetGas, want: uint64(5000)},
		"SstoreClearGas":                     {param: params.SstoreClearGas, want: uint64(5000)},
		"SstoreRefundGas":                    {param: params.SstoreRefundGas, want: uint64(15000)},
		"NetSstoreNoopGas":                   {param: params.NetSstoreNoopGas, want: uint64(200)},
		"NetSstoreInitGas":                   {param: params.NetSstoreInitGas, want: uint64(20000)},
		"NetSstoreCleanGas":                  {param: params.NetSstoreCleanGas, want: uint64(5000)},
		"NetSstoreDirtyGas":                  {param: params.NetSstoreDirtyGas, want: uint64(200)},
		"NetSstoreClearRefund":               {param: params.NetSstoreClearRefund, want: uint64(15000)},
		"NetSstoreResetRefund":               {param: params.NetSstoreResetRefund, want: uint64(4800)},
		"NetSstoreResetClearRefund":          {param: params.NetSstoreResetClearRefund, want: uint64(19800)},
		"SstoreSentryGasEIP2200":             {param: params.SstoreSentryGasEIP2200, want: uint64(2300)},
		"SstoreSetGasEIP2200":                {param: params.SstoreSetGasEIP2200, want: uint64(20000)},
		"SstoreResetGasEIP2200":              {param: params.SstoreResetGasEIP2200, want: uint64(5000)},
		"SstoreClearsScheduleRefundEIP2200":  {param: params.SstoreClearsScheduleRefundEIP2200, want: uint64(15000)},
		"ColdAccountAccessCostEIP2929":       {param: params.ColdAccountAccessCostEIP2929, want: uint64(2600)},
		"ColdSloadCostEIP2929":               {param: params.ColdSloadCostEIP2929, want: uint64(2100)},
		"WarmStorageReadCostEIP2929":         {param: params.WarmStorageReadCostEIP2929, want: uint64(100)},
		"SstoreClearsScheduleRefundEIP3529":  {param: params.SstoreClearsScheduleRefundEIP3529, want: uint64(5000 - 2100 + 1900)},
		"JumpdestGas":                        {param: params.JumpdestGas, want: uint64(1)},
		"EpochDuration":                      {param: params.EpochDuration, want: uint64(30000)},
		"CreateDataGas":                      {param: params.CreateDataGas, want: uint64(200)},
		"CallCreateDepth":                    {param: params.CallCreateDepth, want: uint64(1024)},
		"ExpGas":                             {param: params.ExpGas, want: uint64(10)},
		"LogGas":                             {param: params.LogGas, want: uint64(375)},
		"CopyGas":                            {param: params.CopyGas, want: uint64(3)},
		"StackLimit":                         {param: params.StackLimit, want: uint64(1024)},
		"TierStepGas":                        {param: params.TierStepGas, want: uint64(0)},
		"LogTopicGas":                        {param: params.LogTopicGas, want: uint64(375)},
		"CreateGas":                          {param: params.CreateGas, want: uint64(32000)},
		"Create2Gas":                         {param: params.Create2Gas, want: uint64(32000)},
		"SelfdestructRefundGas":              {param: params.SelfdestructRefundGas, want: uint64(24000)},
		"MemoryGas":                          {param: params.MemoryGas, want: uint64(3)},
		"TxDataNonZeroGasFrontier":           {param: params.TxDataNonZeroGasFrontier, want: uint64(68)},
		"TxDataNonZeroGasEIP2028":            {param: params.TxDataNonZeroGasEIP2028, want: uint64(16)},
		"TxAccessListAddressGas":             {param: params.TxAccessListAddressGas, want: uint64(2400)},
		"TxAccessListStorageKeyGas":          {param: params.TxAccessListStorageKeyGas, want: uint64(1900)},
		"CallGasFrontier":                    {param: params.CallGasFrontier, want: uint64(40)},
		"CallGasEIP150":                      {param: params.CallGasEIP150, want: uint64(700)},
		"BalanceGasFrontier":                 {param: params.BalanceGasFrontier, want: uint64(20)},
		"BalanceGasEIP150":                   {param: params.BalanceGasEIP150, want: uint64(400)},
		"BalanceGasEIP1884":                  {param: params.BalanceGasEIP1884, want: uint64(700)},
		"ExtcodeSizeGasFrontier":             {param: params.ExtcodeSizeGasFrontier, want: uint64(20)},
		"ExtcodeSizeGasEIP150":               {param: params.ExtcodeSizeGasEIP150, want: uint64(700)},
		"SloadGasFrontier":                   {param: params.SloadGasFrontier, want: uint64(50)},
		"SloadGasEIP150":                     {param: params.SloadGasEIP150, want: uint64(200)},
		"SloadGasEIP1884":                    {param: params.SloadGasEIP1884, want: uint64(800)},
		"SloadGasEIP2200":                    {param: params.SloadGasEIP2200, want: uint64(800)},
		"ExtcodeHashGasConstantinople":       {param: params.ExtcodeHashGasConstantinople, want: uint64(400)},
		"ExtcodeHashGasEIP1884":              {param: params.ExtcodeHashGasEIP1884, want: uint64(700)},
		"SelfdestructGasEIP150":              {param: params.SelfdestructGasEIP150, want: uint64(5000)},
		"ExpByteFrontier":                    {param: params.ExpByteFrontier, want: uint64(10)},
		"ExpByteEIP158":                      {param: params.ExpByteEIP158, want: uint64(50)},
		"ExtcodeCopyBaseFrontier":            {param: params.ExtcodeCopyBaseFrontier, want: uint64(20)},
		"ExtcodeCopyBaseEIP150":              {param: params.ExtcodeCopyBaseEIP150, want: uint64(700)},
		"CreateBySelfdestructGas":            {param: params.CreateBySelfdestructGas, want: uint64(25000)},
		"DefaultBaseFeeChangeDenominator":    {param: params.DefaultBaseFeeChangeDenominator, want: 8},
		"DefaultElasticityMultiplier":        {param: params.DefaultElasticityMultiplier, want: 2},
		"InitialBaseFee":                     {param: params.InitialBaseFee, want: 1000000000},
		"MaxCodeSize":                        {param: params.MaxCodeSize, want: 24576},
		"MaxInitCodeSize":                    {param: params.MaxInitCodeSize, want: 2 * 24576},
		"EcrecoverGas":                       {param: params.EcrecoverGas, want: uint64(3000)},
		"Sha256BaseGas":                      {param: params.Sha256BaseGas, want: uint64(60)},
		"Sha256PerWordGas":                   {param: params.Sha256PerWordGas, want: uint64(12)},
		"Ripemd160BaseGas":                   {param: params.Ripemd160BaseGas, want: uint64(600)},
		"Ripemd160PerWordGas":                {param: params.Ripemd160PerWordGas, want: uint64(120)},
		"IdentityBaseGas":                    {param: params.IdentityBaseGas, want: uint64(15)},
		"IdentityPerWordGas":                 {param: params.IdentityPerWordGas, want: uint64(3)},
		"Bn256AddGasByzantium":               {param: params.Bn256AddGasByzantium, want: uint64(500)},
		"Bn256AddGasIstanbul":                {param: params.Bn256AddGasIstanbul, want: uint64(150)},
		"Bn256ScalarMulGasByzantium":         {param: params.Bn256ScalarMulGasByzantium, want: uint64(40000)},
		"Bn256ScalarMulGasIstanbul":          {param: params.Bn256ScalarMulGasIstanbul, want: uint64(6000)},
		"Bn256PairingBaseGasByzantium":       {param: params.Bn256PairingBaseGasByzantium, want: uint64(100000)},
		"Bn256PairingBaseGasIstanbul":        {param: params.Bn256PairingBaseGasIstanbul, want: uint64(45000)},
		"Bn256PairingPerPointGasByzantium":   {param: params.Bn256PairingPerPointGasByzantium, want: uint64(80000)},
		"Bn256PairingPerPointGasIstanbul":    {param: params.Bn256PairingPerPointGasIstanbul, want: uint64(34000)},
		"Bls12381G1AddGas":                   {param: params.Bls12381G1AddGas, want: uint64(600)},
		"Bls12381G1MulGas":                   {param: params.Bls12381G1MulGas, want: uint64(12000)},
		"Bls12381G2AddGas":                   {param: params.Bls12381G2AddGas, want: uint64(4500)},
		"Bls12381G2MulGas":                   {param: params.Bls12381G2MulGas, want: uint64(55000)},
		"Bls12381PairingBaseGas":             {param: params.Bls12381PairingBaseGas, want: uint64(115000)},
		"Bls12381PairingPerPairGas":          {param: params.Bls12381PairingPerPairGas, want: uint64(23000)},
		"Bls12381MapG1Gas":                   {param: params.Bls12381MapG1Gas, want: uint64(5500)},
		"Bls12381MapG2Gas":                   {param: params.Bls12381MapG2Gas, want: uint64(110000)},
		"RefundQuotient":                     {param: params.RefundQuotient, want: uint64(2)},
		"RefundQuotientEIP3529":              {param: params.RefundQuotientEIP3529, want: uint64(5)},
		"BlobTxBytesPerFieldElement":         {param: params.BlobTxBytesPerFieldElement, want: 32},
		"BlobTxFieldElementsPerBlob":         {param: params.BlobTxFieldElementsPerBlob, want: 4096},
		"BlobTxBlobGasPerBlob":               {param: params.BlobTxBlobGasPerBlob, want: 1 << 17},
		"BlobTxMinBlobGasprice":              {param: params.BlobTxMinBlobGasprice, want: 1},
		"BlobTxBlobGaspriceUpdateFraction":   {param: params.BlobTxBlobGaspriceUpdateFraction, want: 3338477},
		"BlobTxPointEvaluationPrecompileGas": {param: params.BlobTxPointEvaluationPrecompileGas, want: 50000},
		"BlobTxTargetBlobGasPerBlock":        {param: params.BlobTxTargetBlobGasPerBlock, want: 3 * 131072},
		"MaxBlobGasPerBlock":                 {param: params.MaxBlobGasPerBlock, want: 6 * 131072},
		"GenesisDifficulty":                  {param: params.GenesisDifficulty.Int64(), want: int64(131072)},
		"BeaconRootsStorageAddress":          {param: params.BeaconRootsStorageAddress, want: common.HexToAddress("0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02")},
		"SystemAddress":                      {param: params.SystemAddress, want: common.HexToAddress("0xfffffffffffffffffffffffffffffffffffffffe")},
	}

	for name, test := range tests {
		assert.Equal(t, test.want, test.param, name)
	}
}
