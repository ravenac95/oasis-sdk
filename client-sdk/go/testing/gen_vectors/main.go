// gen_vectors generates test vectors for the staking transactions.
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"

	signature "github.com/oasisprotocol/oasis-sdk/client-sdk/go/crypto/signature"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/helpers"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/modules/consensusaccounts"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/testing"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/types"

	"github.com/oasisprotocol/oasis-core/go/common/quantity"
)

func main() {
	var vectors []helpers.TestVector

	// Generate different gas fees.
	for _, fee := range []*types.Fee{
		{},
		{Amount: types.NewBaseUnits(*quantity.NewFromUint64(10_000_000_000), "_"), Gas: 1000},
		{Amount: types.NewBaseUnits(*quantity.NewFromUint64(0), "_"), Gas: 2000},
		{Amount: types.NewBaseUnits(*quantity.NewFromUint64(424_242_424_242), "ROSE"), Gas: 3000},
		{Amount: types.NewBaseUnits(*quantity.NewFromUint64(123_456_789), "TEST"), Gas: 4000},
	} {
		// Generate different nonces.
		for _, nonce := range []uint64{0, 1, 10, 42, 1000, 1_000_000, 10_000_000, math.MaxUint64} {
			// Prepare transaction.
			for _, amt := range []uint64{0, 1000, 10_000_000, 10_000_000_000_000, 10_000_000_000_000_000_000} {
				for _, w := range []testing.TestKey{testing.Alice, testing.Dave} {
					for _, chainContext := range []signature.Context{
						"53852332637bacb61b91b6411ab4095168ba02a50be4c3f82448438826f23898",
						"5ba68bc5e01e06f755c4c044dd11ec508e4c17f1faf40c0e67874388437a9e55",
					} {
						var tx *types.Transaction
						addr := w.Address.String()
						if len(w.EthAddress) > 0 {
							addr = hex.EncodeToString(w.EthAddress[:])
						}
						depositWithdrawDst, _ := helpers.ResolveAddress(nil, addr)
						tx = consensusaccounts.NewDepositTx(fee, &consensusaccounts.Deposit{
							To:     depositWithdrawDst,
							Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), ""),
						})
						vectors = append(vectors, helpers.MakeTestVector("Deposit", tx, true, w, nonce, chainContext))

						tx = consensusaccounts.NewWithdrawTx(fee, &consensusaccounts.Withdraw{
							To:     depositWithdrawDst,
							Amount: types.NewBaseUnits(*quantity.NewFromUint64(amt), ""),
						})
						vectors = append(vectors, helpers.MakeTestVector("Withdraw", tx, true, w, nonce, chainContext))

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
