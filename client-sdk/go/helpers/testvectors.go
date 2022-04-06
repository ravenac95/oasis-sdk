package helpers

import (
	"encoding/json"
	"log"

	"github.com/oasisprotocol/oasis-core/go/common/cbor"
	"github.com/oasisprotocol/oasis-core/go/common/crypto/hash"
	signature "github.com/oasisprotocol/oasis-sdk/client-sdk/go/crypto/signature"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/testing"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/types"
)

const (
	keySeedPrefix    = "oasis-sdk runtime test vectors: "
	chainContextSeed = "staking test vectors"
)

var chainContext hash.Hash

// TestVector is an Oasis transaction test vector.
type TestVector struct {
	Kind            string            `json:"kind"`
	ChainContext    string            `json:"signature_context"`
	Tx              interface{}       `json:"tx"`
	SignedTx        types.Transaction `json:"signed_tx"`
	EncodedTx       []byte            `json:"encoded_tx"`
	EncodedSignedTx []byte            `json:"encoded_signed_tx"`
	// Valid indicates whether the transaction is (statically) valid.
	// NOTE: This means that the transaction passes basic static validation, but
	// it may still not be valid on the given network due to invalid nonce,
	// or due to some specific parameters set on the network.
	Valid            bool                `json:"valid"`
	SignerPrivateKey []byte              `json:"signer_private_key"`
	SignerPublicKey  signature.PublicKey `json:"signer_public_key"`
}

func init() {
	chainContext.FromBytes([]byte(chainContextSeed))
}

// MakeTestVector generates a new test vector from a transaction using a specific signer.
func MakeTestVector(kind string, tx *types.Transaction, valid bool, w testing.TestKey, nonce uint64, chainContext signature.Context) TestVector {
	tx.AppendAuthSignature(w.SigSpec, nonce)

	// Configure chain context for all signatures using chain domain separation.
	ts := tx.PrepareForSigning()
	if err := ts.AppendSign(chainContext, w.Signer); err != nil {
		log.Panicf("failed to sign transaction: %w", err)
	}

	ut := ts.UnverifiedTransaction()
	txSigned, err := ut.Verify(chainContext)
	if err != nil {
		log.Panicf("failed to verify transaction: %w", err)
	}
	err = txSigned.ValidateBasic()
	if err != nil {
		log.Panicf("failed to validate transaction: %w", err)
	}

	// TODO
	/*bodyType := tx.Call.Method
	v := reflect.New(reflect.TypeOf(bodyType)).Interface()
	if err = cbor.Unmarshal(tx.Call.Body, v); err != nil {
		panic(err)
	}*/

	prettyTx, err := json.Marshal(tx)
	if err != nil {
		panic(err)
	}

	return TestVector{
		Kind:             keySeedPrefix + kind,
		ChainContext:     string(chainContext),
		Tx:               prettyTx,
		SignedTx:         *txSigned,
		EncodedTx:        cbor.Marshal(tx),
		EncodedSignedTx:  cbor.Marshal(txSigned),
		Valid:            valid,
		SignerPrivateKey: []byte{}, // TODO
		SignerPublicKey:  w.Signer.Public(),
	}
}
