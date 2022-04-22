package main

import (
	"fmt"
	"log"

	"github.com/oasisprotocol/oasis-core/go/common/cbor"
	"github.com/oasisprotocol/oasis-core/go/common/crypto/hash"
	"github.com/oasisprotocol/oasis-core/go/common/quantity"
	"github.com/oasisprotocol/oasis-sdk/cli/cmd/common"
	"github.com/oasisprotocol/oasis-sdk/cli/wallet"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/config"
	signature "github.com/oasisprotocol/oasis-sdk/client-sdk/go/crypto/signature"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/helpers"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/modules/consensusaccounts"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/testing"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/types"
)

const (
	keySeedPrefix    = "oasis-sdk runtime test vectors: "
	chainContextSeed = "staking test vectors"
)

var chainContext hash.Hash

// RuntimeTestVector is an Oasis runtime transaction test vector.
type RuntimeTestVector struct {
	Kind         string            `json:"kind"`
	ChainContext string            `json:"chain_context"/`
	Tx           types.Transaction `json:"tx"`
	TxBody       interface{}       `json:"tx_body"`
	// TxDetails stores other tx-specific information which need
	// to be checked, but are not part of the signed transaction.
	// e.g. ethereum address for deposits. User needs to see the ethereum-formatted
	// address and Ledger needs to check, if it really maps into Tx.Body.To.
	TxDetails       interface{}                 `json:"tx_details"`
	SignedTx        types.UnverifiedTransaction `json:"signed_tx"`
	EncodedTx       []byte                      `json:"encoded_tx"`
	EncodedSignedTx []byte                      `json:"encoded_signed_tx"`
	// Valid indicates whether the transaction is (statically) valid.
	// NOTE: This means that the transaction passes basic static validation, but
	// it may still not be valid on the given network due to invalid nonce,
	// or due to some specific parameters set on the network.
	Valid bool `json:"valid"`
	// SignerAlgorithm is AlgorithmEd25519Raw or AlgorithmSecp256k1Raw
	SignerAlgorithm  string              `json:"signer_algorithm"`
	SignerPrivateKey []byte              `json:"signer_private_key"`
	SignerPublicKey  signature.PublicKey `json:"signer_public_key"`
}

func init() {
	chainContext.FromBytes([]byte(chainContextSeed))
}

// MakeRuntimeTestVector generates a new test vector from a transaction using a specific signer.
func MakeRuntimeTestVector(kind string, tx *types.Transaction, txDetails interface{}, valid bool, w testing.TestKey, nonce uint64, chainContext signature.Context) RuntimeTestVector {
	signerAlgorithm := wallet.AlgorithmEd25519Raw
	if w.SigSpec.Secp256k1Eth != nil {
		signerAlgorithm = wallet.AlgorithmSecp256k1Raw
	}

	gasLimit := uint64(100000)
	gasPrice := uint64(100001)

	npw := common.NPWSelection{
		NetworkName: "mainnet",
		Network: &config.Network{
			ChainContext: string(chainContext),
		},
		ParaTimeName: "emerald",
		ParaTime: &config.ParaTime{
			Description: "emerald",
			ID:          "000000000000000000000000000000000000000000000000e2eaa99fc008f87f",
			Denominations: map[string]*config.DenominationInfo{
				"ROSE": {
					Symbol:   "ROSE",
					Decimals: 18,
				},
			},
		},
	}

	sigTx := SignParaTimeTransaction(&npw, w, tx, nonce, gasLimit, gasPrice)

	// TODO: Use introspecion to figure out which
	//txBody := reflect.New(reflect.TypeOf(tx.Call.Body)).Interface()
	txBody := consensusaccounts.Deposit{}
	if err := cbor.Unmarshal(tx.Call.Body, &txBody); err != nil {
		log.Fatalf("error unmarhalling body: %v", err)
	}
	return RuntimeTestVector{
		Kind:             keySeedPrefix + kind,
		ChainContext:     string(chainContext),
		Tx:               *tx,
		TxBody:           txBody,
		TxDetails:        txDetails,
		SignedTx:         *sigTx,
		EncodedTx:        cbor.Marshal(tx),
		EncodedSignedTx:  cbor.Marshal(sigTx),
		Valid:            valid,
		SignerAlgorithm:  signerAlgorithm,
		SignerPrivateKey: w.UnsafeBytes,
		SignerPublicKey:  w.Signer.Public(),
	}
}

// SignParaTimeTransaction signs a ParaTime transaction.
func SignParaTimeTransaction(
	npw *common.NPWSelection,
	w testing.TestKey,
	tx *types.Transaction,
	nonce uint64,
	txGasLimit uint64,
	txGasPrice uint64,
) *types.UnverifiedTransaction {
	// Default to passed values and do online estimation when possible.
	tx.AuthInfo.Fee.Gas = txGasLimit

	gasPrice := &types.BaseUnits{}
	// TODO: Support different denominations for gas fees.
	var err error
	gasPrice, err = helpers.ParseParaTimeDenomination(npw.ParaTime, fmt.Sprintf("%d", txGasPrice), types.NativeDenomination)
	if err != nil {
		log.Fatalf("bad gas price: %w", err)
	}

	// Prepare the transaction before (optional) gas estimation to ensure correct estimation.
	tx.AppendAuthSignature(w.SigSpec, nonce)

	// Compute fee amount based on gas price.
	if err := gasPrice.Amount.Mul(quantity.NewFromUint64(tx.AuthInfo.Fee.Gas)); err != nil {
		log.Fatalf("error computing gasPrice: %w", err)
	}
	tx.AuthInfo.Fee.Amount.Amount = gasPrice.Amount
	tx.AuthInfo.Fee.Amount.Denomination = gasPrice.Denomination

	// TODO: Support confidential transactions (only in online mode).

	// Sign the transaction.
	sigCtx := signature.DeriveChainContext(npw.ParaTime.Namespace(), npw.Network.ChainContext)
	ts := tx.PrepareForSigning()
	if err := ts.AppendSign(sigCtx, w.Signer); err != nil {
		log.Fatalf("failed to sign transaction: %w", err)
	}

	return ts.UnverifiedTransaction()
}
