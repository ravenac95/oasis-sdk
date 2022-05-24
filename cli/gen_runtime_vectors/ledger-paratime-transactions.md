# Design doc: Ledger support for ParaTime transactions

This document describes the [format of ParaTime transactions](#runtime-transaction-format),
[APDU changes](#apduspec-changes) and new UI/UX on the Ledger devices for
signing the ParaTime transactions:
1. [deposit, withdrawal and transfer transactions](#signing-deposit-withdrawal-and-transfer-transactions),
2. [encrypted transactions](#signing-encrypted-transactions),
3. [unencrypted contract transactions](#signing-unencrypted-contracts-transactions),
4. [EVM transactions](#signing-evm-transactions).

## Runtime transaction format

The structure of the runtime (ParaTime) transaction to be signed by Ledger is
the following:

```go
// Transaction is a runtime transaction.
type Transaction struct {
	cbor.Versioned

	Call     Call     `json:"call"`
	AuthInfo AuthInfo `json:"ai"`
}
```

The transaction **can be signed either with `Secp256k1` ("Ethereum") or
`ed25519` key!** Information on this + fee is stored inside `ai`
field. Consult the [golang code][client-sdk ai] for details.

`call` is defined as [follows][client-sdk call]:

```go
// Call is a method call.
type Call struct {
	Format CallFormat      `json:"format,omitempty"`
	Method string          `json:"method,omitempty"`
	Body   cbor.RawMessage `json:"body"`
}
```

If `format` equals `0`, the transaction is unencrypted and it is CBOR-encoded
inside the `body` field.

If `format` equals `1`, the transaction is encrypted. In this case `method` is
empty and `body` contains a CBOR-encoded `CallEnvelopeX25519DeoxysII` which
includes the encrypted transaction body inside the `data` field.

[client-sdk tx]: https://github.com/oasisprotocol/oasis-sdk/blob/ea88627b4db9fc48c71a96d96a5cece08a259c6d/client-sdk/go/types/transaction.go#L158-L164
[client-sdk ai]: https://github.com/oasisprotocol/oasis-sdk/blob/ea88627b4db9fc48c71a96d96a5cece08a259c6d/client-sdk/go/types/transaction.go#L258-L262
[client-sdk call]: https://github.com/oasisprotocol/oasis-sdk/blob/ea88627b4db9fc48c71a96d96a5cece08a259c6d/client-sdk/go/types/transaction.go#L251-L256

## APDUSPEC changes

### GET_ADDR_SR25519

#### Command

| Field      | Type           | Content                | Expected       |
| ---------- | -------------- | ---------------------- | -------------- |
| CLA        | byte (1)       | Application Identifier | 0x05           |
| INS        | byte (1)       | Instruction ID         | 0x03           |
| P1         | byte (1)       | Request User confirmation | No = 0      |
| P2         | byte (1)       | Parameter 2            | ignored        |
| L          | byte (1)       | Bytes in payload       | (depends)      |
| Path[0]    | byte (4)       | Derivation Path Data   | 44             |
| Path[1]    | byte (4)       | Derivation Path Data   | 474            |
| Path[2]    | byte (4)       | Derivation Path Data   | ?              |
| Path[3]    | byte (4)       | Derivation Path Data   | ?              |
| Path[4]    | byte (4)       | Derivation Path Data   | ?              |

First three items in the derivation path will be hardened automatically hardened

#### Response

| Field   | Type      | Content               | Note                     |
| ------- | --------- | --------------------- | ------------------------ |
| PK      | byte (32) | Public Key            |                          |
| ADDR    | byte (??) | Bech 32 addr          |                          |
| SW1-SW2 | byte (2)  | Return code           | see list of return codes |

### GET_ADDR_SECP256K1

#### Command

| Field      | Type           | Content                | Expected       |
| ---------- | -------------- | ---------------------- | -------------- |
| CLA        | byte (1)       | Application Identifier | 0x05           |
| INS        | byte (1)       | Instruction ID         | 0x04           |
| P1         | byte (1)       | Request User confirmation | No = 0      |
| P2         | byte (1)       | Parameter 2            | ignored        |
| L          | byte (1)       | Bytes in payload       | (depends)      |
| Path[0]    | byte (4)       | Derivation Path Data   | 44             |
| Path[1]    | byte (4)       | Derivation Path Data   | 0              |
| Path[2]    | byte (4)       | Derivation Path Data   | ?              |
| Path[3]    | byte (4)       | Derivation Path Data   | ?              |
| Path[4]    | byte (4)       | Derivation Path Data   | ?              |

First three items in the derivation path will be hardened automatically hardened

#### Response

| Field   | Type      | Content               | Note                     |
| ------- | --------- | --------------------- | ------------------------ |
| PK      | byte (32) | Public Key            |                          |
| ADDR    | byte (??) | Hex addr              |                          |
| SW1-SW2 | byte (2)  | Return code           | see list of return codes |

### SIGN_PT_ED25519

#### Command

| Field | Type     | Content                | Expected  |
| ----- | -------- | ---------------------- | --------- |
| CLA   | byte (1) | Application Identifier | 0x05      |
| INS   | byte (1) | Instruction ID         | 0x05      |
| P1    | byte (1) | Payload desc           | 0 = init  |
|       |          |                        | 1 = add   |
|       |          |                        | 2 = last  |
| P2    | byte (1) | ----                   | not used  |
| L     | byte (1) | Bytes in payload       | (depends) |

The first packet/chunk includes only the derivation path

All other packets/chunks should contain message to sign

*First Packet*

| Field      | Type     | Content                | Expected  |
| ---------- | -------- | ---------------------- | --------- |
| Path[0]    | byte (4) | Derivation Path Data   | 44        |
| Path[1]    | byte (4) | Derivation Path Data   | 474       |
| Path[2]    | byte (4) | Derivation Path Data   | ?         |
| Path[3]    | byte (4) | Derivation Path Data   | ?         |
| Path[4]    | byte (4) | Derivation Path Data   | ?         |

*Other Chunks/Packets*

| Field   | Type     | Content              | Expected |
| ------- | -------- | -------------------- | -------- |
| Data    | bytes... | Context+Message+Meta |          |

Data is defined as:

| Field   | Type     | Content                    | Expected     |
| ------- | -------- | -------------------------- | ------------ |
| CtxLen  | byte     | Context Length             |              |
| Context | bytes..  | Context                    | CtxLen bytes |
| Message | bytes..  | CBOR data to sign          |              |
| Meta    | bytes..  | CBOR metadata to verify    |              |

#### Response

| Field   | Type      | Content     | Note                     |
| ------- | --------- | ----------- | ------------------------ |
| SIG     | byte (64) | Signature   |                          |
| SW1-SW2 | byte (2)  | Return code | see list of return codes |

### SIGN_PT_SR25519

#### Command

| Field | Type     | Content                | Expected  |
| ----- | -------- | ---------------------- | --------- |
| CLA   | byte (1) | Application Identifier | 0x05      |
| INS   | byte (1) | Instruction ID         | 0x06      |
| P1    | byte (1) | Payload desc           | 0 = init  |
|       |          |                        | 1 = add   |
|       |          |                        | 2 = last  |
| P2    | byte (1) | ----                   | not used  |
| L     | byte (1) | Bytes in payload       | (depends) |

The first packet/chunk includes only the derivation path

All other packets/chunks should contain message to sign

*First Packet*

| Field      | Type     | Content                | Expected  |
| ---------- | -------- | ---------------------- | --------- |
| Path[0]    | byte (4) | Derivation Path Data   | 44        |
| Path[1]    | byte (4) | Derivation Path Data   | 474       |
| Path[2]    | byte (4) | Derivation Path Data   | ?         |
| Path[3]    | byte (4) | Derivation Path Data   | ?         |
| Path[4]    | byte (4) | Derivation Path Data   | ?         |

*Other Chunks/Packets*

| Field   | Type     | Content              | Expected |
| ------- | -------- | -------------------- | -------- |
| Data    | bytes... | Context+Message+Meta |          |

Data is defined as:

| Field   | Type     | Content                    | Expected     |
| ------- | -------- | -------------------------- | ------------ |
| CtxLen  | byte     | Context Length             |              |
| Context | bytes..  | Context                    | CtxLen bytes |
| Message | bytes..  | CBOR data to sign          |              |
| Meta    | bytes..  | CBOR metadata to verify    |              |

#### Response

| Field   | Type      | Content     | Note                     |
| ------- | --------- | ----------- | ------------------------ |
| SIG     | byte (64) | Signature   |                          |
| SW1-SW2 | byte (2)  | Return code | see list of return codes |

### SIGN_PT_SECP256K1

#### Command

| Field | Type     | Content                | Expected  |
| ----- | -------- | ---------------------- | --------- |
| CLA   | byte (1) | Application Identifier | 0x05      |
| INS   | byte (1) | Instruction ID         | 0x07      |
| P1    | byte (1) | Payload desc           | 0 = init  |
|       |          |                        | 1 = add   |
|       |          |                        | 2 = last  |
| P2    | byte (1) | ----                   | not used  |
| L     | byte (1) | Bytes in payload       | (depends) |

The first packet/chunk includes only the derivation path

All other packets/chunks should contain message to sign

*First Packet*

| Field      | Type     | Content                | Expected  |
| ---------- | -------- | ---------------------- | --------- |
| Path[0]    | byte (4) | Derivation Path Data   | 44        |
| Path[1]    | byte (4) | Derivation Path Data   | 0         |
| Path[2]    | byte (4) | Derivation Path Data   | ?         |
| Path[3]    | byte (4) | Derivation Path Data   | ?         |
| Path[4]    | byte (4) | Derivation Path Data   | ?         |

*Other Chunks/Packets*

| Field   | Type     | Content              | Expected |
| ------- | -------- | -------------------- | -------- |
| Data    | bytes... | Context+Message+Meta |          |

Data is defined as:

| Field   | Type     | Content                    | Expected     |
| ------- | -------- | -------------------------- | ------------ |
| CtxLen  | byte     | Context Length             |              |
| Context | bytes..  | Context                    | CtxLen bytes |
| Message | bytes..  | CBOR data to sign          |              |
| Meta    | bytes..  | CBOR metadata to verify    |              |

#### Response

| Field   | Type      | Content     | Note                     |
| ------- | --------- | ----------- | ------------------------ |
| SIG     | byte (64) | Signature   |                          |
| SW1-SW2 | byte (2)  | Return code | see list of return codes |

## Signing deposit, withdrawal and transfer transactions

### Deposit (+Allowance)

Depositing to ParaTime consists of two transactions. First, we need to submit
the (consensus) allowance transaction. This is already supported by Ledger and
we propose the following improvement:

```ledger
|     Type     > | <    To    > | <   Amount   > | <     Fee     > | < Gas limit > | <  Network  > | <             > | <              |
|   Allowance    |     <TO>     |   +-<AMOUNT>   |    <FEE> ROSE   |  <GAS LIMIT>  |   <NETWORK>   |     APPROVE     |     REJECT     |
|                |              |      ROSE      |                 |               |               |                 |                |
```

**ENHANCEMENT OF EXISTING ALLOWANCE UI:** If `NETWORK` matches Mainnet or
Testnet, Ledger should render the following in place of `TO` for specific
addresses:
- `oasis1qrnu9yhwzap7rqh6tdcdcpz0zf86hwhycchkhvt8`: `Cipher Mainnet`
- `oasis1qqdn25n5a2jtet2s5amc7gmchsqqgs4j0qcg5k0t`: `Cipher Testnet`
- `oasis1qzvlg0grjxwgjj58tx2xvmv26era6t2csqn22pte`: `Emerald Mainnet`
- `oasis1qr629x0tg9gm5fyhedgs9lw5eh3d8ycdnsxf0run`: `Emerald Testnet`

The addresses above should be hardcoded into Oasis Ledger App. If you
are interested in how these were derived from the ParaTime ID check the
[staking document].

The second step is to sign the `consensus.Deposit` transaction with the
following proposed Ledger UI:

```ledger
|     Type     > | <   To (1/1)  > | <   Amount    > | <     Fee     > | < Gas limit > | <  Network  > | <  ParaTime  > | <             > | <               |
|    Deposit     |    <MIXED_TO>   |  <AMOUNT> ROSE  |    <FEE> ROSE   |  <GAS LIMIT>  |   <NETWORK>   |  <RUNTIME ID>  |     APPROVE     |      REJECT     |
|   (ParaTime)   |                 |                 |                 |               |               |                |                 |                 |
```

`MIXED_TO` can either be `oasis1` or the Ethereum's `0x` address. If `Meta`
does not contain `orig_to` field, render the `tx.call.body.to` value in
`oasis1` format in place of `MIXED_TO`. If `Meta` contains `orig_to` field,
then:
1. Check that the `0x` address stored in `orig_to` field maps to the `oasis1`
   address of `tx.call.body.to` according to this [mapping function].
2. Render `orig_to` value in `0x` format in place of `MIXED_TO`.

If `tx.call.body.to` is empty, then the deposit is made to the signer's account
inside the ParaTime. In this case `(transaction signer)` literal is rendered in
place of `MIXED_TO`.

**NOTE:** Ledger should store `NETWORK, RUNTIME ID, DENOMINATION -> NUMBER OF
DECIMALS` pairs for known ParaTimes (Emerald, Cipher), so that it correctly
renders `AMOUNT` and `FEE`. If the denomination is not known, Ledger
shows the value in base units without decimal point and symbol denomination
(although the symbol is provided in the `types.BaseUnits` object).

`RUNTIME ID` shows the 32-byte hex encoded ParaTime ID. If the `NETWORK`
matches Mainnet or Testnet, then human-readable version of the `RUNTIME ID` is
shown for specific values:
- `000000000000000000000000000000000000000000000000e199119c992377cb`: `Cipher Mainnet`
- `0000000000000000000000000000000000000000000000000000000000000000`: `Cipher Testnet`
- `000000000000000000000000000000000000000000000000e2eaa99fc008f87f`: `Emerald Mainnet`
- `00000000000000000000000000000000000000000000000072c8215e60d5bca7`: `Emerald Testnet`

`RUNTIME ID` and `NETWORK` (also called chain domain separation context)
cannot be derived from the signature context as was the case for consensus
transactions. For ParaTime transactions they are contained in `Meta`. Each time
Ledger needs to construct a signature context by using the `runtime_id` and
`chain_context` fields stored in `Meta` and verify that it matches `Context`.
See [golang implementation][chain context function] for details.

[staking document]: https://docs.oasis.dev/oasis-core/consensus/services/staking/#runtime-accounts
[mapping function]: https://github.com/oasisprotocol/oasis-sdk/blob/e566b326ab1c34f3d811b50f96c53c3a79a91826/client-sdk/go/types/address.go#L134-L141
[chain context function]: https://github.com/oasisprotocol/oasis-sdk/blob/e566b326ab1c34f3d811b50f96c53c3a79a91826/client-sdk/go/crypto/signature/context.go

### Withdraw

The `consensus.Withdraw` transaction has the following Ledger UI:

```ledger
|     Type     > | <   To (1/1)  > | <   Amount    > | <     Fee     > | < Gas limit > | <  Network  > | <  ParaTime  > | <             > | <               |
|   Withdraw     |       <TO>      |  <AMOUNT> ROSE  |    <FEE> ROSE   |  <GAS LIMIT>  |   <NETWORK>   |  <RUNTIME ID>  |     APPROVE     |      REJECT     |
|  (ParaTime)    |                 |                 |                 |               |               |                |                 |                 |
```

If `tx.call.body.to` is empty, then the withdrawal is made to the signer's
consensus account. In this case `(transaction signer)` literal is rendered in
place of `TO`.

### Transfer

The `accounts.Transfer` transaction has the following Ledger UI:

```ledger
|     Type     > | <   To (1/1)  > | <   Amount    > | <     Fee     > | < Gas limit > | <  Network  > | <  ParaTime  > | <             > | <               |
|   Transfer     |    <MIXED_TO>   |  <AMOUNT> ROSE  |    <FEE> ROSE   |  <GAS LIMIT>  |   <NETWORK>   |  <RUNTIME ID>  |     APPROVE     |      REJECT     |
|  (ParaTime)    |                 |                 |                 |               |               |                |                 |                 |
```

### Example

The user wants to deposit 100 ROSE to `0x90adE3B7065fa715c7a150313877dF1d33e777D5`
account on Emerald ParaTime on the Mainnet. First they sign the deposit
allowance transaction for the Emerald ParaTime.

```ledger
|     Type     > | <    To    > | <   Amount   > | < Gas limit > | <     Fee     > | <  Network  > | <             > | <              |
|   Allowance    |   Emerald    |  +100.00 ROSE  |     1277      |     0.0 ROSE    |    Mainnet    |     APPROVE     |     REJECT     |
|                |   Mainnet    |                |               |                 |               |                 |                |
```

Next, they sign the ParaTime deposit transaction.

```ledger
|     Type     > | <   To (1/2)  > | <    To (2/2)   > | <   Amount    > | <     Fee     > | < Gas limit > | <  Network  > | <  ParaTime  > | <             > | <               |
|    Deposit     | 0x90adE3B7065fa | dF1d33e777D5      |   100.00 ROSE   |    0.00 ROSE    |     11310     |    Mainnet    |     Emerald    |     APPROVE     |      REJECT     |
|   (ParaTime)   | 715c7a150313877 |                   |                 |                 |               |               |                |                 |                 |
```

Then, they transfer some tokens to another account inside the ParaTime:

```ledger
|     Type     > | <    To (1/2)  > | <    To (2/2)   > | <   Amount    > | <     Fee     > | < Gas limit > | <  Network  > | <  ParaTime  > | <             > | <               |
|   Transfer     | oasis1qpupfu7e2n | m8anj64ytrayne    |   10.00 ROSE    |   0.00015 ROSE  |     11311     |    Mainnet    |    Emerald     |     APPROVE     |      REJECT     |
|  (ParaTime)    | 6pkezeaw0yhj8mce |                   |                 |                 |               |               |                |                 |                 |
```

Finally, the user withdraws the remainder of tokens back to the Mainnet.

```ledger
|     Type     > | <    To (1/2)  > | <    To (2/2)   > | <   Amount    > | <     Fee     > | < Gas limit > | <  Network  > | <  ParaTime  > | <             > | <               |
|   Withdraw     | oasis1qrec770vre | 504k68svq7kzve    |  99.99970 ROSE  |   0.00015 ROSE  |     11311     |    Mainnet    |    Emerald     |     APPROVE     |      REJECT     |
|  (ParaTime)    | k0a9a5lcrv0zvt22 |                   |                 |                 |               |               |                |                 |                 |
```

## Signing encrypted transactions

To sign encrypted transactions (`tx.call.format != 0`) the user needs to enable
**Blind signing** in the Ledger Settings first. Ledger will then render the
following UI:

```ledger
|   Review  > | < BLIND > | <    Fee    > | <  Network  > | <  ParaTime > | <            > | <             |
|  Encrypted  |  SIGNING! |   <FEE> ROSE  |   <NETWORK>   |  <RUNTIME ID> |    APPROVE     |     REJECT    |
| Transaction |           |               |               |               |                |               |
```

## Signing unencrypted contracts transactions

### Uploading smart contract

`contracts.Upload` transaction will not be signed by Ledger because the size
of the WASM byte code may easily exceed the maximum size of Ledger transaction.

<!-- We do not sign complete contracts anymore
```ledger
| Review Contract > | < Contract hash (1/2) > | < Contract hash (2/2) > | < ParaTime ID > | <    Fee    > | < Genesis Hash (1/2) > | < Genesis Hash (2/2) > | <             > | <               |
|     Upload        | a8fc73270dff2bbd2bc7a15 | 6b69847e90b782e781      |     Cipher      |   0.0 ROSE    | 53852332637bacb61b91b6 | c3f82448438826f23898   |     APPROVE     |      REJECT     |
|                   | cf4c1ec6375e6deefc5f2d5 |                         |    (Testnet)    |               | 411ab4095168ba02a50be4 |                        |                 |                 |
```
-->

### Instantiating smart contract

`contracts.Instantiate` shows the following UI on Ledger:

```ledger
|  Review Contract > | < Code ID > | < Amount (1/1) > | < Function > | < Argument 1 (1/1) > | ... | <    Fee    > | < Gas limit > | <  Network  > | <  ParaTime  > | <             > | <               |
|   Instantiation    |  <CODE ID>  |   <AMOUNT...>    |  <FUNCTION>  |      <ARGUMENT>      | ... |  <FEE> ROSE   |  <GAS LIMIT>  |   <NETWORK>   |  <RUNTIME ID>  |     APPROVE     |      REJECT     |
|                    |             |                  |              |                      | ... |               |               |               |                |                 |                 |
```

`FUNCTION` and `ARGUMENT` are extracted from `tx.call.body.data["instantiate"]`
as the key name of the first item and the value respectively. If
`tx.call.body.data["instantiate"]` is empty or not present, then Function and
Argument screens are hidden.

`AMOUNT...` is the amount of tokens sent. Contract SDK supports sending
multiple tokens at once, each with its own denomination symbol. Ledger should
render all of them, one per page. For rendering rules of each `AMOUNT` consult
the [Deposit (+Allowance)](#deposit-allowance) behavior.

There can be multiple Argument screens `Argument 1`, `Argument 2`, ...,
`Argument N` for each function argument. `ARGUMENT` can be one of the
following types:
- string
- number (integer, float)
- array
- map
- boolean
- null

Strings are rendered as UTF-8 strings and the following characters need to be
escaped: `:`, `,`, `}`, `]`, `…`.

Numbers are rendered in standard general base-10 encoding using decimal period.
For strings and numbers that cannot fit a single page, pagination is introduced.

Boolean and null values are rendered as `true`, `false` and `null` respectively
on a single page.

Array and map is rendered in form `VAL1,VAL2,...` and `KEY1:VAL1,KEY1:VAL1,...`
respectively. For security, **the items of the map must be sorted
lexicographically by KEY**. `KEY` and `VAL` can be of any supported type. If it
is a map or array it is rendered as `{ARGUMENT}` or `[ARGUMENT]` respectively
to avoid disambiguation. Otherwise, it is just `ARGUMENT`.

If the content of an array or a map cannot fit a single page, no pagination
is introduced. Instead, the content is trimmed, ellipsis `…` is appended at
the end and the screen **becomes confirmable**. If the user double-clicks it, a
subscreen for item `n` of an array or a map is shown. There is one subscreen
for each item of the array or a map of size `N` titled `Argument n.1`,
`Argument n.2`, ..., `Argument n.N` which renders the item `n` as
`ARGUMENT` for an array item or `ARGUMENT:ARGUMENT` for a map item:

```ledger
|   Argument 1.1 (1/1) > | < Argument 1.2 (1/1) | < Argument 1.3 (1/1) | ... | <          |
|       <ARGUMENT>       |      <ARGUMENT>      |      <ARGUMENT>      |     |    BACK    |
|                        |                      |                      |     |            |
```

The recursive approach described above allows user to browse through a complete
tree of function arguments with ⬅️ and ➡️ buttons, visit a child by double-
clicking and returning to a parent node by confirming the _BACK_ screen.

The maximum string length, the length of the array, the depth of a map have
reasonable limits on Ledger. If the limit is exceeded, Ledger displays an error
on the initial screen. Then, if the user still wants to sign such a
transaction, they need to enable **Blind signing** in the Settings menu. The
following UI is shown when blind-signing a non-encrypted transaction.

```ledger
|  Review Contract > | < BLIND > | < Instance ID (1/1) > | <   Amount    > | <     Fee     > | <  Network  > | <  ParaTime > | <            > | <             |
|    Instantiate     |  SIGNING! |     <INSTANCE ID>     |  <AMOUNT> ROSE  |   <FEE> ROSE    |   <NETWORK>   |  <RUNTIME ID> |    APPROVE     |     REJECT    |
|                    |           |                       |                 |                 |               |               |                |               |
```

### Calling smart contract

Ledger should show details of the ParaTime transaction to the user, when this is
possible. `contracts.Call` shows the following UI on Ledger:

```ledger
| Review Contract > | < Instance ID > | < Amount (1/1) > | < Function > | < Argument 1 (1/1) > | ... | <     Fee     > | < Gas limit > | <  Network  > | <  ParaTime  > | <             > | <              |
|      Call         |   <INSTANCE ID> |   <AMOUNT...>    |  <FUNCTION>  |      <ARGUMENT>      | ... |   <FEE> ROSE    |  <GAS LIMIT>  |   <NETWORK>   |   <RUNTIME ID> |     APPROVE     |     REJECT     |
|                   |                 |                  |              |                      | ... |                 |               |               |                |                 |                |
```

The behavior of UI is based on the UI for `contracts.Instantiate` transaction.
`FUNCTION` and `ARGUMENT` are extracted from `tx.call.body.data["call"]`
as the key name of the first item and the value respectively.

### Upgrading smart contracts

Signing `contracts.Upgrade` shows the following UI on Ledger:

```ledger
|  Review Contract > | < Instance ID (1/1) > | < Amount (1/1) > | < New Code ID (1/1) > | < Preupgrade args (1/1) > | ... | < Postupgrade args (1/1) | ... | < ParaTime ID (1/1) > | <     Fee     > | < Gas limit > | < Network > | < ParaTime > | <             > | <               |
|      Upgrade       |     <INSTANCE ID>     |   <AMOUNT...>    |      <CODE_ID>        |         <ARGUMENT>        |     |       <ARGUMENT>         |     |     <RUNTIME ID>      |   <FEE> ROS     |  <GAS LIMIT>  |  <NETWORK>  | <RUNTIME ID> |    APPROVE      |      REJECT     |
|                    |                       |                  |                       |                           |     |                          |     |                       |                 |               |             |              |                 |                 |
```

The behavior of UI is based on the UI for `contracts.Instantiate` transaction.
`FUNCTION` and `ARGUMENT` are extracted from `tx.call.body.data["pre_upgrade"]`
and `tx.call.body.data["post_upgrade"]` for Preupgrade and Postupgrade screens
respectively. If a particular value is missing, hide the corresponding screen.

### Example

To upload, instantiate and call the [hello world example] running on Testnet
Cipher ParaTime the user first signs the contract upload transaction with a
file-based ed25519 keypair. The user obtains the `Code ID` 3 for the uploaded
contract.

Next, the user instantiates the contract and obtains the `Instance ID` 2.

```ledger
|  Review Contract > | < Code ID > | <  Amount  > | <  Function  > | <  Argument 1  > | <    Fee    > | < Gas limit > | <  Network  > | <  ParaTime  > | <             > | <               |
|   Instantiation    |      3      |   0.0 ROSE   |  instantiate   | initial_counter: |    0.0 ROSE   |     1348      |    Mainnet    |     Cipher     |     APPROVE     |      REJECT     |
|                    |             |              |                |        42        |               |               |               |                |                 |                 |
```

Finally, they perform a call to `say_hello` function on a smart contract
passing the `{"who":"me"}` object as a function argument.

```ledger
| Review Contract > | < Instance ID > | <  Amount  > | < Function > | < Argument 1 > | <     Fee     > | < Gas limit > | <  Network  > | <  ParaTime  > | <             > | <              |
|      Call         |       2         |   0.0 ROSE   |  say_hello   |     who:me     |     0.0 ROSE    |     1283      |    Mainnet    |     Cipher     |     APPROVE     |     REJECT     |
|                   |                 |              |              |                |                 |               |               |                |                 |                |
```

The user can also provide a more complex object as an argument:

```json
{
  "who": {
    "username": "alice",
    "client_secret": "e5868ebb4445fc2ad9f949956c1cb9ddefa0d421",
    "last_logins": [1646835046, 1615299046, 1583763046, 1552140646],
    "redirect": null
  }
}
```

In this case the Ledger generates the following UI.

```ledger
| Review Contract > | <  Instance ID  > | <   Amount  > | < Function > | <   Argument 1   > | <     Fee     > | < Gas limit > | <  Network  > | <  ParaTime  > | <            > | <              |
|      Call         |       2           |    0.0 ROSE   |   say_hello  | who:{username:alic |    0.15 ROSE    |     1283      |    Mainnet    |     Cipher     |    APPROVE     |     REJECT     |
|                   |                   |               |              | e,client_secret:e… |                 |               |               |                |                |                |

                                                                       V                    V

                                                                       |      Arg. 1.1    > | < Arg. 1.2 (1/2) > | < Arg. 1.2 (2/2) > | <    Arg. 1.3    > | <   Arg 1.4    > | <            |
                                                                       |   username:alice   | client_secret:e586 | 956c1cb9ddefa0d421 | last_logins:[16468 |   redirect:null  |     BACK     |
                                                                       |                    | 8ebb4445fc2ad9f949 |                    | 35046,1615299046,… |                  |              |

                                                                                                                                      V                    V

                                                                                                                                      |     Arg. 1.3.1   > | <   Arg. 1.3.2   > | <   Arg. 1.3.3   > | <   Arg. 1.3.4     | <              |
                                                                                                                                      |     1646835046     |     1615299046     |     1583763046     |     1552140646     |      BACK      |
                                                                                                                                      |                    |                    |                    |                    |                |
```

[hello world example]: https://docs.oasis.dev/oasis-sdk/contract/hello-world#deploying-the-contract

## Signing EVM transactions

### Creating smart contract

`evm.Create` transaction will not be managed by Ledger because the size of the
WASM byte code may easily exceed the maximum size of Ledger transaction.

### Calling smart contract

In contrast to `contracts.Call`, `evm.Call` requires contract ABI to extract
feasible argument names from the RLP-encoded transaction. We do not support
this, so only **blind signing** is performed which the user needs to enable in
the Ledger Settings first. The UI is as follows:

```ledger
|  Review EVM > | < BLIND > | < Address (1/1) > | <   Amount    > | <     Fee     > | <  Network  > | <  ParaTime > | <            > | <             |
|   Contract    |  SIGNING! |     <ADDRESS>     |  <AMOUNT> ROSE  |   <FEE> ROSE    |   <NETWORK>   |  <RUNTIME ID> |    APPROVE     |     REJECT    |
|    Call       |           |                   |                 |                 |               |               |                |               |
```

<!--## Signing generic SubmitMsg transactions

TODO

There will most likely be new ParaTime transactions which need to be signed by
Ledger in the future. Could we make a generic UI for any ParaTime transaction
(e.g. delete contract) so we wouldn't need to release a new version
of Oasis Ledger App each time?

`roothash.SubmitMsg` transaction will replace the
`staking.Allow` and runtime's deposit transaction. See [ADR 11] for details.

Some references:
- https://github.com/oasisprotocol/oasis-sdk/blob/main/runtime-sdk/src/types/transaction.rs
- https://github.com/oasisprotocol/oasis-sdk/blob/main/runtime-sdk/modules/contracts/src/types.rs

[ADR 11]: https://docs.oasis.dev/oasis-core/adr/0011-incoming-runtime-messages

## Ledger-related Open Questions

1. Currently, we never show information where the tokens were
   sent/deposited/withdrawn **from**. We noticed the same in Ethereum Ledger
   App. Isn't there a security issue, that the app could pick a wrong account
   ID used to send the tokens and the user wouldn't know it?
   Should we add to our UI:
   - the from address for all Oasis transactions,
   - the originating genesis hash and ParaTime ID for all cross-chain
     transactions?
2. ParaTime Deposit requires two transactions (allowance + deposit).
   Could we simplify the UI on Ledger by batching them and signing them both in
   a single user intervention? Or is double-click mandatory to access the Ledger's
   private key each time?
3. Would there be any issues with Ledger when parsing CBOR to
   build the proposed tree-like menu UI for browsing through provided smart
   contract arguments?
4. In the example above we avoided signing smart contract upload
   transactions, because we assumed Ledger is not fast enough in practise to
   sign such large transactions (e.g. signing 500kB of data with ed25519 scheme
   under 10 seconds). Is this a valid assumption?
-->
## Footnotes

### Signing contract uploads with Ledger

In the future perhaps, if only the merkle root hash of the WASM
contract would be contained in the transaction, signing such a contract could
be feasible. See how Ethereum 2.x contract deployment is done using this
approach.
