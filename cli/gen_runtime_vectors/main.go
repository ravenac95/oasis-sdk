// gen_runtime_vectors generates test vectors for runtime transactions.
package main

import (
	"encoding/json"
	"fmt"
	"log"
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

var (
	toAddresses = []string{
		"0x90adE3B7065fa715c7a150313877dF1d33e777D5",     // Dave
		"oasis1qpupfu7e2n6pkezeaw0yhj8mcem8anj64ytrayne", // Dave
		"oasis1qrec770vrek0a9a5lcrv0zvt22504k68svq7kzve", // Alice
	}
)

func main() {
	var vectors []RuntimeTestVector

	// Prepare signature context.
	var rtId common.Namespace
	rtId.UnmarshalHex(config.DefaultNetworks.All["mainnet"].ParaTimes.All["emerald"].ID)
	rtIdHex, err := rtId.MarshalHex()
	if err != nil {
		log.Panic(err)
	}

	// Generate different gas fees.
	for _, fee := range []*types.Fee{
		{},
		{Amount: types.NewBaseUnits(*quantity.NewFromUint64(0), "_"), Gas: 2000},
		{Amount: types.NewBaseUnits(*quantity.NewFromUint64(424_242_424_242), "ROSE"), Gas: 3000},
		{Amount: types.NewBaseUnits(*quantity.NewFromUint64(123_456_789), "TEST"), Gas: 4000},
	} {
		// Generate different nonces.
		for _, nonce := range []uint64{0, 1, 42, math.MaxUint64} {
			// Prepare transaction.
			for _, amt := range []uint64{0, 1000, 10_000_000_000_000_000_000} {
				for _, w := range []testing.TestKey{testing.Alice, testing.Dave} {
					for _, addr := range toAddresses {
						for _, chainContext := range []signature.Context{
							"53852332637bacb61b91b6411ab4095168ba02a50be4c3f82448438826f23898",
							"5ba68bc5e01e06f755c4c044dd11ec508e4c17f1faf40c0e67874388437a9e55",
						} {
							sigCtx := signature.DeriveChainContext(rtId, string(chainContext))
							var tx *types.Transaction
							var meta map[string]string
							var txBody interface{}

							depositWithdrawDst, _ := helpers.ResolveAddress(nil, addr)
							txBody = &consensusaccounts.Deposit{
								To:     depositWithdrawDst,
								Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
							}
							tx = consensusaccounts.NewDepositTx(fee, txBody.(*consensusaccounts.Deposit))
							meta = map[string]string{
								"orig_to":       addr,
								"runtime_id":    string(rtIdHex),
								"chain_context": string(chainContext),
							}
							vectors = append(vectors, MakeRuntimeTestVector("Deposit", tx, txBody, meta, true, w, nonce, sigCtx))

							txBody = &consensusaccounts.Withdraw{
								To:     depositWithdrawDst,
								Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), "ROSE"),
							}
							tx = consensusaccounts.NewWithdrawTx(fee, txBody.(*consensusaccounts.Withdraw))
							meta = map[string]string{
								"runtime_id":    string(rtIdHex),
								"chain_context": string(chainContext),
							}
							vectors = append(vectors, MakeRuntimeTestVector("Withdraw", tx, txBody, meta, true, w, nonce, sigCtx))

						}
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
