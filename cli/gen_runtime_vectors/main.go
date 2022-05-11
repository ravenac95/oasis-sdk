// gen_runtime_vectors generates test vectors for runtime transactions.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/oasisprotocol/oasis-core/go/common"
	"github.com/oasisprotocol/oasis-core/go/common/quantity"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/config"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/crypto/signature"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/helpers"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/modules/consensusaccounts"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/testing"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/types"
)

const (
	aliceNativeAddr = "oasis1qrec770vrek0a9a5lcrv0zvt22504k68svq7kzve"
	daveEthAddr     = "0x90adE3B7065fa715c7a150313877dF1d33e777D5"
	daveNativeAddr  = "oasis1qpupfu7e2n6pkezeaw0yhj8mcem8anj64ytrayne"
	unknownEthAddr  = "0x4ad80CBfBFe645BACCe3504166EF38aA5C15a35f"

	// Invalid runtime ID for signature context.
	unknownRtIdHex = "8000000000000000000000000000000000000000000000000000000001234567"

	// Invalid chain context.
	unknownChainContext = "abcdef01234567890ea817cc1446c401752a05a249b36c9b9876543210fedcba"
)

func main() {
	var vectors []RuntimeTestVector

	// Valid runtime ID for signature context.
	rtIdHex := config.DefaultNetworks.All["mainnet"].ParaTimes.All["emerald"].ID
	var rtId common.Namespace
	rtId.UnmarshalHex(rtIdHex)

	var tx *types.Transaction
	var meta map[string]string
	var txBody interface{}

	for _, fee := range []*types.Fee{
		{},
		{Amount: types.NewBaseUnits(*quantity.NewFromUint64(0), "_"), Gas: 2000},
		{Amount: types.NewBaseUnits(*quantity.NewFromUint64(424_242_424_242), "ROSE"), Gas: 3000},
		{Amount: types.NewBaseUnits(*quantity.NewFromUint64(123_456_789), "TEST"), Gas: 4000},
	} {
		for _, nonce := range []uint64{0, 1, math.MaxUint64} {
			for _, amt := range []uint64{0, 1000, 10_000_000_000_000_000_000} {
				for _, chainContext := range []signature.Context{
					signature.Context(config.DefaultNetworks.All["mainnet"].ChainContext),
					signature.Context(config.DefaultNetworks.All["testnet"].ChainContext),
				} {
					sigCtx := signature.DeriveChainContext(rtId, string(chainContext))
					var dst *types.Address

					// Valid Deposit: Alice -> Alice account on ParaTime
					txBody = &consensusaccounts.Deposit{
						To:     nil,
						Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
					}
					tx = consensusaccounts.NewDepositTx(fee, txBody.(*consensusaccounts.Deposit))
					meta = map[string]string{
						"runtime_id":    rtIdHex,
						"chain_context": string(chainContext),
					}
					vectors = append(vectors, MakeRuntimeTestVector("Deposit", tx, txBody, meta, true, testing.Alice, nonce, sigCtx))

					// Valid Deposit: Alice -> Dave's native account on ParaTime
					dst, _ = helpers.ResolveAddress(nil, daveNativeAddr)
					txBody = &consensusaccounts.Deposit{
						To:     dst,
						Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
					}
					tx = consensusaccounts.NewDepositTx(fee, txBody.(*consensusaccounts.Deposit))
					meta = map[string]string{
						"runtime_id":    rtIdHex,
						"chain_context": string(chainContext),
					}
					vectors = append(vectors, MakeRuntimeTestVector("Deposit", tx, txBody, meta, true, testing.Alice, nonce, sigCtx))

					// Valid Deposit: Alice -> Dave's ethereum account on ParaTime
					dst, _ = helpers.ResolveAddress(nil, daveEthAddr)
					txBody = &consensusaccounts.Deposit{
						To:     dst,
						Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
					}
					tx = consensusaccounts.NewDepositTx(fee, txBody.(*consensusaccounts.Deposit))
					meta = map[string]string{
						"orig_to":       daveEthAddr,
						"runtime_id":    rtIdHex,
						"chain_context": string(chainContext),
					}
					vectors = append(vectors, MakeRuntimeTestVector("Deposit", tx, txBody, meta, true, testing.Alice, nonce, sigCtx))

					// Invalid Deposit: orig_to doesn't match transaction's to
					dst, _ = helpers.ResolveAddress(nil, daveEthAddr)
					txBody = &consensusaccounts.Deposit{
						To:     dst,
						Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
					}
					tx = consensusaccounts.NewDepositTx(fee, txBody.(*consensusaccounts.Deposit))
					meta = map[string]string{
						"orig_to":       unknownEthAddr,
						"runtime_id":    rtIdHex,
						"chain_context": string(chainContext),
					}
					vectors = append(vectors, MakeRuntimeTestVector("Deposit", tx, txBody, meta, false, testing.Alice, nonce, sigCtx))

					// Invalid Deposit: runtime_id doesn't match the one in sigCtx
					dst, _ = helpers.ResolveAddress(nil, daveEthAddr)
					txBody = &consensusaccounts.Deposit{
						To:     dst,
						Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
					}
					tx = consensusaccounts.NewDepositTx(fee, txBody.(*consensusaccounts.Deposit))
					meta = map[string]string{
						"orig_to":       daveEthAddr,
						"runtime_id":    unknownRtIdHex,
						"chain_context": string(chainContext),
					}
					vectors = append(vectors, MakeRuntimeTestVector("Deposit", tx, txBody, meta, false, testing.Alice, nonce, sigCtx))

					// Invalid Deposit: chain_context doesn't match the one in sigCtx
					dst, _ = helpers.ResolveAddress(nil, daveEthAddr)
					txBody = &consensusaccounts.Deposit{
						To:     dst,
						Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
					}
					tx = consensusaccounts.NewDepositTx(fee, txBody.(*consensusaccounts.Deposit))
					meta = map[string]string{
						"orig_to":       daveEthAddr,
						"runtime_id":    rtIdHex,
						"chain_context": unknownChainContext,
					}
					vectors = append(vectors, MakeRuntimeTestVector("Deposit", tx, txBody, meta, false, testing.Alice, nonce, sigCtx))

					// Valid Withdraw: Alice -> Alice account on consensus
					txBody = &consensusaccounts.Withdraw{
						To:     nil,
						Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
					}
					tx = consensusaccounts.NewWithdrawTx(fee, txBody.(*consensusaccounts.Withdraw))
					meta = map[string]string{
						"runtime_id":    rtIdHex,
						"chain_context": string(chainContext),
					}
					vectors = append(vectors, MakeRuntimeTestVector("Withdraw", tx, txBody, meta, true, testing.Alice, nonce, sigCtx))

					// Valid Withdraw: Alice -> Dave account on consensus
					dst, _ = helpers.ResolveAddress(nil, daveNativeAddr)
					txBody = &consensusaccounts.Withdraw{
						To:     dst,
						Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
					}
					tx = consensusaccounts.NewWithdrawTx(fee, txBody.(*consensusaccounts.Withdraw))
					meta = map[string]string{
						"runtime_id":    rtIdHex,
						"chain_context": string(chainContext),
					}
					vectors = append(vectors, MakeRuntimeTestVector("Withdraw", tx, txBody, meta, true, testing.Alice, nonce, sigCtx))

					// Valid Withdraw: Dave secp256k1 account -> Dave address on consensus
					txBody = &consensusaccounts.Withdraw{
						To:     nil,
						Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
					}
					tx = consensusaccounts.NewWithdrawTx(fee, txBody.(*consensusaccounts.Withdraw))
					meta = map[string]string{
						"runtime_id":    rtIdHex,
						"chain_context": string(chainContext),
					}
					vectors = append(vectors, MakeRuntimeTestVector("Withdraw", tx, txBody, meta, true, testing.Dave, nonce, sigCtx))

					// Valid Withdraw: Dave secp256k1 account -> Alice account on consensus
					dst, _ = helpers.ResolveAddress(nil, aliceNativeAddr)
					txBody = &consensusaccounts.Withdraw{
						To:     dst,
						Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
					}
					tx = consensusaccounts.NewWithdrawTx(fee, txBody.(*consensusaccounts.Withdraw))
					meta = map[string]string{
						"runtime_id":    rtIdHex,
						"chain_context": string(chainContext),
					}
					vectors = append(vectors, MakeRuntimeTestVector("Withdraw", tx, txBody, meta, true, testing.Dave, nonce, sigCtx))

					// Invalid Withdraw: runtime_id doesn't match the one in sigCtx
					dst, _ = helpers.ResolveAddress(nil, aliceNativeAddr)
					txBody = &consensusaccounts.Withdraw{
						To:     dst,
						Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
					}
					tx = consensusaccounts.NewWithdrawTx(fee, txBody.(*consensusaccounts.Withdraw))
					meta = map[string]string{
						"runtime_id":    unknownRtIdHex,
						"chain_context": string(chainContext),
					}
					vectors = append(vectors, MakeRuntimeTestVector("Withdraw", tx, txBody, meta, false, testing.Dave, nonce, sigCtx))

					// Invalid Withdraw: chain_context doesn't match the one in sigCtx
					dst, _ = helpers.ResolveAddress(nil, aliceNativeAddr)
					txBody = &consensusaccounts.Withdraw{
						To:     dst,
						Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
					}
					tx = consensusaccounts.NewWithdrawTx(fee, txBody.(*consensusaccounts.Withdraw))
					meta = map[string]string{
						"runtime_id":    rtIdHex,
						"chain_context": unknownChainContext,
					}
					vectors = append(vectors, MakeRuntimeTestVector("Withdraw", tx, txBody, meta, false, testing.Dave, nonce, sigCtx))
				}
			}
		}
	}

	// Generate output.
	jsonOut, err := json.MarshalIndent(&vectors, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding test vectors: %v\n", err)
	}
	fmt.Printf("%s", jsonOut)
}
