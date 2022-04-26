package main

import (
	"log"

	"github.com/oasisprotocol/oasis-core/go/common"
	"github.com/oasisprotocol/oasis-core/go/common/cbor"
	"github.com/oasisprotocol/oasis-core/go/common/crypto/hash"
	"github.com/oasisprotocol/oasis-sdk/cli/wallet"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/config"
	signature "github.com/oasisprotocol/oasis-sdk/client-sdk/go/crypto/signature"
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
	ChainContext string            `json:"chain_context"`
	RuntimeId    common.Namespace  `json:"runtime_id"`
	SigCtx       string            `json:"signature_ctx"`
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
func MakeRuntimeTestVector(kind string, tx *types.Transaction, txDetails interface{}, valid bool, w testing.TestKey, nonce uint64, rtId common.Namespace, chainContext signature.Context) RuntimeTestVector {
	signerAlgorithm := wallet.AlgorithmEd25519Raw
	if w.SigSpec.Secp256k1Eth != nil {
		signerAlgorithm = wallet.AlgorithmSecp256k1Raw
	}

	// Prepare the transaction before (optional) gas estimation to ensure correct estimation.
	tx.AppendAuthSignature(w.SigSpec, nonce)

	// TODO: Support confidential transactions (only in online mode).

	// Sign the transaction.
	rtIdHex, err := rtId.MarshalHex()
	if err != nil {
		log.Fatalf("error marshalling runtime id: %v", err)
	}
	pt := &config.ParaTime{
		ID: string(rtIdHex),
	}
	sigCtx := signature.DeriveChainContext(pt.Namespace(), string(chainContext))
	ts := tx.PrepareForSigning()
	if err := ts.AppendSign(sigCtx, w.Signer); err != nil {
		log.Fatalf("failed to sign transaction: %w", err)
	}

	sigTx := ts.UnverifiedTransaction()

	// TODO: Use introspecion to figure out which
	//txBody := reflect.New(reflect.TypeOf(tx.Call.Body)).Interface()
	txBody := consensusaccounts.Deposit{}
	if err := cbor.Unmarshal(tx.Call.Body, &txBody); err != nil {
		log.Fatalf("error unmarhalling body: %v", err)
	}
	return RuntimeTestVector{
		Kind:             keySeedPrefix + kind,
		ChainContext:     string(chainContext),
		RuntimeId:        rtId,
		SigCtx:           string(sigCtx),
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
