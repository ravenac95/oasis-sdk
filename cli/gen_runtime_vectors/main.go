// gen_runtime_vectors generates test vectors for runtime transactions.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/oasisprotocol/oasis-core/go/common"
	"github.com/oasisprotocol/oasis-core/go/common/quantity"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/config"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/crypto/signature"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/helpers"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/modules/accounts"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/modules/consensusaccounts"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/modules/contracts"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/modules/evm"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/testing"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/types"
)

const (
	// TODO: Derive addresses below from testKeys directly.
	aliceNativeAddr = "oasis1qrec770vrek0a9a5lcrv0zvt22504k68svq7kzve"
	daveEthAddr     = "0x90adE3B7065fa715c7a150313877dF1d33e777D5"
	daveNativeAddr  = "oasis1qpupfu7e2n6pkezeaw0yhj8mcem8anj64ytrayne"
	eveEthAddr      = "0xFe94510049b95A8BfD7D6397177d7D2e2E5201Aa"
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

	for _, fee := range []*types.Fee{
		{},
		{Amount: types.NewBaseUnits(*quantity.NewFromUint64(0), "_"), Gas: 2000},
		{Amount: types.NewBaseUnits(*quantity.NewFromUint64(424_242_424_242), "ROSE"), Gas: 3000},
		{Amount: types.NewBaseUnits(*quantity.NewFromUint64(123_456_789), "TEST"), Gas: 4000},
	} {
		for _, nonce := range []uint64{0, 1, math.MaxUint64} {
			for _, chainContext := range []signature.Context{
				signature.Context(config.DefaultNetworks.All["mainnet"].ChainContext),
				signature.Context(config.DefaultNetworks.All["testnet"].ChainContext),
			} {
				sigCtx := signature.DeriveChainContext(rtId, string(chainContext))

				for _, amt := range []uint64{0, 1000, 10_000_000_000_000_000_000} {
					// consensusaccounts.Deposit
					for _, t := range []struct {
						to           string
						origTo       string
						rtId         string
						chainContext string
						valid        bool
					}{
						// Valid Deposit: Alice -> Alice account on ParaTime
						{"", "", rtIdHex, string(chainContext), true},
						// Valid Deposit: Alice -> Dave's native account on ParaTime
						{daveNativeAddr, "", rtIdHex, string(chainContext), true},
						// Valid Deposit: Alice -> Dave's ethereum account on ParaTime
						{daveEthAddr, daveEthAddr, rtIdHex, string(chainContext), true},
						// Valid Deposit: Alice -> Dave's ethereum account on ParaTime
						{daveEthAddr, daveEthAddr, rtIdHex, string(chainContext), true},
						// Valid Deposit: Alice -> Dave's ethereum account on ParaTime, lowercased
						{daveEthAddr, strings.ToLower(daveEthAddr), rtIdHex, string(chainContext), true},
						// Valid Deposit: Alice -> Dave's ethereum account on ParaTime without 0x
						{daveEthAddr, daveEthAddr[2:], rtIdHex, string(chainContext), true},
						// Valid Deposit: Alice -> Dave's ethereum account on ParaTime, lowercase without 0x
						{daveEthAddr, strings.ToLower(daveEthAddr[2:]), rtIdHex, string(chainContext), true},
						// Invalid Deposit: orig_to doesn't match transaction's to
						{daveEthAddr, unknownEthAddr, rtIdHex, string(chainContext), false},
						// Invalid Deposit: runtime_id doesn't match the one in sigCtx
						{daveEthAddr, daveEthAddr, unknownRtIdHex, string(chainContext), false},
						// Invalid Deposit: chain_context doesn't match the one in sigCtx
						{daveEthAddr, daveEthAddr, rtIdHex, unknownChainContext, false},
					} {
						to, _ := helpers.ResolveAddress(nil, t.to)
						txBody := &consensusaccounts.Deposit{
							To:     to,
							Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
						}
						tx = consensusaccounts.NewDepositTx(fee, txBody)
						meta = map[string]string{
							"runtime_id":    t.rtId,
							"chain_context": t.chainContext,
						}
						if t.origTo != "" {
							meta["orig_to"] = t.origTo
						}
						vectors = append(vectors, MakeRuntimeTestVector(tx, txBody, meta, t.valid, testing.Alice, nonce, sigCtx))
					}

					// consensusaccounts.Withdraw
					for _, t := range []struct {
						to           string
						signer       testing.TestKey
						rtId         string
						chainContext string
						valid        bool
					}{
						// Valid Withdraw: Alice -> Alice account on consensus
						{"", testing.Alice, rtIdHex, string(chainContext), true},
						// Valid Withdraw: Alice -> Dave account on consensus
						{daveNativeAddr, testing.Alice, rtIdHex, string(chainContext), true},
						// Valid Withdraw: Dave secp256k1 account -> Dave address on consensus
						{"", testing.Dave, rtIdHex, string(chainContext), true},
						// Valid Withdraw: Dave secp256k1 account -> Alice account on consensus
						{aliceNativeAddr, testing.Dave, rtIdHex, string(chainContext), true},
						// Invalid Withdraw: runtime_id doesn't match the one in sigCtx
						{aliceNativeAddr, testing.Dave, unknownRtIdHex, string(chainContext), false},
						// Invalid Withdraw: chain_context doesn't match the one in sigCtx
						{aliceNativeAddr, testing.Dave, rtIdHex, unknownChainContext, false},
					} {
						to, _ := helpers.ResolveAddress(nil, t.to)
						txBody := &consensusaccounts.Withdraw{
							To:     to,
							Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
						}
						tx = consensusaccounts.NewWithdrawTx(fee, txBody)
						meta = map[string]string{
							"runtime_id":    t.rtId,
							"chain_context": t.chainContext,
						}
						vectors = append(vectors, MakeRuntimeTestVector(tx, txBody, meta, t.valid, t.signer, nonce, sigCtx))
					}

					// accounts.Transfer
					for _, t := range []struct {
						to           string
						origTo       string
						signer       testing.TestKey
						rtId         string
						chainContext string
						valid        bool
					}{
						// Valid Transfer: Alice -> Dave's native account on ParaTime
						{daveNativeAddr, "", testing.Alice, rtIdHex, string(chainContext), true},
						// Valid Transfer: Alice -> Dave's ethereum account on ParaTime
						{daveEthAddr, daveEthAddr, testing.Alice, rtIdHex, string(chainContext), true},
						// Valid Transfer: Dave secp256k1 account -> Alice account on ParaTime
						{aliceNativeAddr, "", testing.Dave, rtIdHex, string(chainContext), true},
						// Valid Transfer: Dave secp256k1 account -> Eve's ethereum account on ParaTime
						{eveEthAddr, eveEthAddr, testing.Dave, rtIdHex, string(chainContext), true},
						// Valid Transfer: Dave secp256k1 account -> Eve's ethereum account on ParaTime, lowercase
						{eveEthAddr, strings.ToLower(eveEthAddr), testing.Dave, rtIdHex, string(chainContext), true},
						// Valid Transfer: Dave secp256k1 account -> Eve's ethereum account on ParaTime, without 0x
						{eveEthAddr, eveEthAddr[2:], testing.Dave, rtIdHex, string(chainContext), true},
						// Valid Transfer: Dave secp256k1 account -> Eve's ethereum account on ParaTime, lowercase without 0x
						{eveEthAddr, strings.ToLower(eveEthAddr[2:]), testing.Dave, rtIdHex, string(chainContext), true},
						// Invalid Transfer: orig_to doesn't match transaction's to
						{daveEthAddr, unknownEthAddr, testing.Alice, rtIdHex, string(chainContext), false},
						// Invalid Transfer: runtime_id doesn't match the one in sigCtx
						{daveEthAddr, daveEthAddr, testing.Alice, unknownRtIdHex, string(chainContext), false},
						// Invalid Transfer: chain_context doesn't match the one in sigCtx
						{daveEthAddr, daveEthAddr, testing.Alice, rtIdHex, unknownChainContext, false},
					} {
						to, _ := helpers.ResolveAddress(nil, t.to)
						txBody := &accounts.Transfer{
							To:     *to,
							Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
						}
						tx = accounts.NewTransferTx(fee, txBody)
						meta = map[string]string{
							"runtime_id":    t.rtId,
							"chain_context": t.chainContext,
						}
						if t.origTo != "" {
							meta["orig_to"] = t.origTo
						}
						vectors = append(vectors, MakeRuntimeTestVector(tx, txBody, meta, t.valid, t.signer, nonce, sigCtx))
					}
				}

				for _, t := range []struct {
					signer       testing.TestKey
					rtId         string
					chainContext string
					valid        bool
				}{
					{testing.Alice, rtIdHex, string(chainContext), true},
					{testing.Dave, rtIdHex, string(chainContext), true},
					{testing.Alice, unknownRtIdHex, string(chainContext), false},
					{testing.Alice, rtIdHex, unknownChainContext, false},
				} {
					for _, tokens := range [][]types.BaseUnits{
						{
							types.BaseUnits{
								Amount:       *quantity.NewFromUint64(1_000_000_000),
								Denomination: "ROSE",
							},
							types.BaseUnits{
								Amount:       *quantity.NewFromUint64(2_000),
								Denomination: "WBTC",
							},
							types.BaseUnits{
								Amount:       *quantity.NewFromUint64(3_000_000),
								Denomination: "WETH",
							},
						},
						{
							types.BaseUnits{
								Amount:       *quantity.NewFromUint64(1_000_000_000_000),
								Denomination: "ROSE",
							},
						},
						{
							types.BaseUnits{
								Amount:       *quantity.NewFromUint64(0),
								Denomination: "TEST",
							},
						},
						{},
					} {
						for _, id := range []uint64{0, 1, math.MaxUint64} {
							// contracts.Call
							txBodyCall := &contracts.Call{
								ID:     contracts.InstanceID(id),
								Data:   nil, // TODO
								Tokens: tokens,
							}
							tx = contracts.NewCallTx(fee, txBodyCall)
							meta = map[string]string{
								"runtime_id":    t.rtId,
								"chain_context": t.chainContext,
							}
							vectors = append(vectors, MakeRuntimeTestVector(tx, txBodyCall, meta, t.valid, t.signer, nonce, sigCtx))

							// contracts.Instantiate
							txBodyInstantiate := &contracts.Instantiate{
								CodeID:         contracts.CodeID(id),
								UpgradesPolicy: contracts.Policy{}, // TODO
								Data:           nil,                // TODO
								Tokens:         tokens,
							}
							tx = contracts.NewInstantiateTx(fee, txBodyInstantiate)
							meta = map[string]string{
								"runtime_id":    t.rtId,
								"chain_context": t.chainContext,
							}
							vectors = append(vectors, MakeRuntimeTestVector(tx, txBodyInstantiate, meta, t.valid, t.signer, nonce, sigCtx))

							// contracts.Upgrade
							txBodyUpgrade := &contracts.Upgrade{
								ID:     contracts.InstanceID(id),
								CodeID: contracts.CodeID(0 ^ id),
								Data:   nil, // TODO
								Tokens: tokens,
							}
							tx = contracts.NewUpgradeTx(fee, txBodyUpgrade)
							meta = map[string]string{
								"runtime_id":    t.rtId,
								"chain_context": t.chainContext,
							}
							vectors = append(vectors, MakeRuntimeTestVector(tx, txBodyUpgrade, meta, t.valid, t.signer, nonce, sigCtx))

							// contracts.Upload not supported by Ledger due to tx bytecode size.
						}
					}

					{
						// evm.Call
						txBodyCall := &evm.Call{
							Address: nil, // TODO
							Value:   nil, // TODO
							Data:    nil, // TODO
						}
						tx = evm.NewCallTx(fee, txBodyCall)
						meta = map[string]string{
							"runtime_id":    t.rtId,
							"chain_context": t.chainContext,
						}
						vectors = append(vectors, MakeRuntimeTestVector(tx, txBodyCall, meta, t.valid, t.signer, nonce, sigCtx))

						// evm.Create not supported by Ledger due to tx bytecode size.
					}
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
