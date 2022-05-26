# 1st Phase Proposal

This document proposes Phase 1 Oasis Ledger App features for the ParaTime transactions.
Complete design document is available [here](ledger-paratime-transactions.md).

## APDUSPEC changes

- [GET_ADDR_SECP256K1](ledger-paratime-transactions-ui.md#get_addr_secp256k1)
- [SIGN_PT_ED25519](ledger-paratime-transactions-ui.md#sign_pt_ed25519)
- [SIGN_PT_SECP256K1](ledger-paratime-transactions-ui.md#sign_pt_secp256k1)
- Implement `Context` verification by checking `Meta.runtime_id` and `Meta.chain_context`.

## Signing deposit, withdrawal and transfer transactions

- Implement `0x` → `oasis1` mapping and verification of `Meta.orig_to`.
- Deposit (+Allowance)
- Withdraw
- Transfer

## Signing encrypted transactions

Complete implementation without _blind signing_ setting - simply assume it
is enabled.

## Signing unencrypted contracts transactions

Blind signing UI for `FUNCTION` and `ARGUMENT` only. Showing `AMOUNT...`
should be fully implemented.

- Instantiating smart contract
- Calling smart contract
- Upgrading smart contracts

## Signing EVM transactions

- Calling smart contract
