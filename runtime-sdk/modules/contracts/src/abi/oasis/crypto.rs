//! Crypto function imports.
use std::convert::TryInto;

use oasis_contract_sdk_crypto as crypto;
use oasis_contract_sdk_types::crypto::SignatureKind;
use oasis_runtime_sdk::{context::Context, crypto::signature};

use super::{memory::Region, OasisV1};
use crate::{
    abi::{gas, ExecutionContext},
    Config, Error,
};

impl<Cfg: Config> OasisV1<Cfg> {
    /// Link crypto helper functions.
    pub fn link_crypto<C: Context>(
        instance: &mut wasm3::Instance<'_, '_, ExecutionContext<'_, C>>,
    ) -> Result<(), Error> {
        // crypto.ecdsa_recover(input) -> response
        let _ = instance.link_function(
            "crypto",
            "ecdsa_recover",
            |ctx, request: ((u32, u32), (u32, u32))| -> Result<(), wasm3::Trap> {
                // Make sure function was called in valid context.
                let ec = ctx.context.ok_or(wasm3::Trap::Abort)?;

                // Charge gas.
                gas::use_gas(ctx.instance, ec.params.gas_costs.wasm_crypto_ecdsa_recover)?;

                ctx.instance
                    .runtime()
                    .try_with_memory(|mut memory| -> Result<_, wasm3::Trap> {
                        let input = Region::from_arg(request.0)
                            .as_slice(&memory)
                            .map_err(|_| wasm3::Trap::Abort)?
                            .to_vec();

                        let output: &mut [u8; 65] = Region::from_arg(request.1)
                            .as_slice_mut(&mut memory)
                            .map_err(|_| wasm3::Trap::Abort)?
                            .try_into()
                            .map_err(|_| wasm3::Trap::Abort)?;

                        let key = crypto::ecdsa::recover(&input).unwrap_or_else(|_| [0; 65]);
                        output.copy_from_slice(&key);

                        Ok(())
                    })?
            },
        );

        // crypto.signature_verify(public_key, context, message, signature) -> response
        #[allow(clippy::type_complexity)]
        let _ = instance.link_function(
            "crypto",
            "signature_verify",
            |ctx,
             (kind, key, context, message, signature): (
                u32,
                (u32, u32),
                (u32, u32),
                (u32, u32),
                (u32, u32),
            )|
             -> Result<u32, wasm3::Trap> {
                // Make sure function was called in valid context.
                let ec = ctx.context.ok_or(wasm3::Trap::Abort)?;

                // Validate message length.
                if message.1 > ec.params.max_crypto_signature_verify_message_size_bytes {
                    return Err(wasm3::Trap::Abort);
                }

                let kind: SignatureKind = kind.try_into().map_err(|_| wasm3::Trap::Abort)?;

                // Charge gas.
                let cost = match kind {
                    SignatureKind::Ed25519 => {
                        ec.params.gas_costs.wasm_crypto_signature_verify_ed25519
                    }
                    SignatureKind::Secp256k1 => {
                        ec.params.gas_costs.wasm_crypto_signature_verify_secp256k1
                    }
                    SignatureKind::Sr25519 => {
                        ec.params.gas_costs.wasm_crypto_signature_verify_sr25519
                    }
                };
                gas::use_gas(ctx.instance, cost)?;

                ctx.instance
                    .runtime()
                    .try_with_memory(|memory| -> Result<_, wasm3::Trap> {
                        let key = get_key(kind, key, &memory)?;
                        let message = Region::from_arg(message)
                            .as_slice(&memory)
                            .map_err(|_| wasm3::Trap::Abort)?;
                        let signature: signature::Signature = Region::from_arg(signature)
                            .as_slice(&memory)
                            .map_err(|_| wasm3::Trap::Abort)?
                            .to_vec()
                            .into();
                        if context.0 != 0
                            && context.1 != 0
                            && matches!(kind, SignatureKind::Sr25519)
                        {
                            let context = Region::from_arg(context)
                                .as_slice(&memory)
                                .map_err(|_| wasm3::Trap::Abort)?;
                            Ok(1 - key.verify(context, message, &signature).is_ok() as u32)
                        } else {
                            Ok(1 - key.verify_raw(message, &signature).is_ok() as u32)
                        }
                    })?
            },
        );

        Ok(())
    }
}

fn get_key(
    kind: SignatureKind,
    key: (u32, u32),
    memory: &wasm3::Memory<'_>,
) -> Result<signature::PublicKey, wasm3::Trap> {
    let region = Region::from_arg(key)
        .as_slice(memory)
        .map_err(|_| wasm3::Trap::Abort)?;

    match kind {
        SignatureKind::Ed25519 => {
            let ed25519 = signature::ed25519::PublicKey::from_bytes(region)
                .map_err(|_| wasm3::Trap::Abort)?;
            Ok(signature::PublicKey::Ed25519(ed25519))
        }
        SignatureKind::Secp256k1 => {
            let secp256k1 = signature::secp256k1::PublicKey::from_bytes(region)
                .map_err(|_| wasm3::Trap::Abort)?;
            Ok(signature::PublicKey::Secp256k1(secp256k1))
        }
        SignatureKind::Sr25519 => {
            let sr25519 = signature::sr25519::PublicKey::from_bytes(region)
                .map_err(|_| wasm3::Trap::Abort)?;
            Ok(signature::PublicKey::Sr25519(sr25519))
        }
    }
}

#[cfg(test)]
mod test {
    extern crate test;
    use super::*;
    use test::Bencher;

    use k256::{self, ecdsa::{self, signature::Verifier as _}};

    const BENCH_CODE: &[u8] = include_bytes!("../../../../../../tests/contracts/bench/target/wasm32-unknown-unknown/release/bench.wasm");
    const MESSAGE: &[u8] = include_bytes!("../../../../../../tests/contracts/bench/data/message.txt");
    const SIGNATURE: &[u8] = include_bytes!("../../../../../../tests/contracts/bench/data/signature.bin");
    const KEY: &[u8] = include_bytes!("../../../../../../tests/contracts/bench/data/key.bin");

    fn verify_signature(message: &[u8], signature: &[u8], key: &[u8]) -> Result<(), ()> {
        let key = k256::EncodedPoint::from_bytes(key).map_err(|_| ())?;
        let sig = ecdsa::Signature::from_der(signature).map_err(|_| ())?;
        let verifying_key = ecdsa::VerifyingKey::from_encoded_point(&key).map_err(|_| ())?;
        verifying_key.verify(message, &sig).map_err(|_| ())?;
        Ok(())
    }

    #[bench]
    fn bench_crypto_nonwasm_plain(b: &mut Bencher) {
        b.iter(|| {
            verify_signature(MESSAGE, SIGNATURE, KEY).unwrap();
        });
    }

    #[bench]
    fn bench_crypto_called_from_wasm_included(b: &mut Bencher) {
        let env = wasm3::Environment::new().expect("creating a new wasm3 environment should succeed");
        let module = env.parse_module(BENCH_CODE).expect("parsing the code should succeed");
        let rt: wasm3::Runtime<'_, wasm3::CallContext<'_, ()>> = env.new_runtime(1 * 1024 * 1024, None).expect("creating a new wasm3 runtime should succeed");
        //let rt = env.new_runtime(1 * 1024 * 1024, None).expect("creating a new wasm3 runtime should succeed");
        let mut instance = rt.load_module(module).expect("instance creation should succeed");
        instance.link_function("bench", "verify_signature", |ctx, (message, signature, key): ((u32, u32), (u32, u32), (u32, u32))| -> Result<(), wasm3::Trap> {
            ctx.instance.runtime().try_with_memory(|memory| -> Result<_, wasm3::Trap> {
                let message = Region::from_arg(message)
                    .as_slice(&memory)
                    .map_err(|_| wasm3::Trap::Abort)?;
                let signature = Region::from_arg(signature)
                    .as_slice(&memory)
                    .map_err(|_| wasm3::Trap::Abort)?;
                let key = Region::from_arg(key)
                    .as_slice(&memory)
                    .map_err(|_| wasm3::Trap::Abort)?;
                verify_signature(message, signature, key).map_err(|_| wasm3::Trap::Abort)?;
                Ok(())
            })?
        });
        let func = instance.find_function::<(), ()>("call_verification_included").expect("finding the entrypoint function should succeed");
        b.iter(|| {
            func.call(()).expect("function call should succeed");
        });
    }

    #[bench]
    fn bench_crypto_computed_in_wasm(b: &mut Bencher) {
        let env = wasm3::Environment::new().expect("creating a new wasm3 environment should succeed");
        let module = env.parse_module(BENCH_CODE).expect("parsing the code should succeed");
        let rt: wasm3::Runtime<'_, wasm3::CallContext<'_, ()>> = env.new_runtime(1 * 1024 * 1024, None).expect("creating a new wasm3 runtime should succeed");
        //let rt = env.new_runtime(1 * 1024 * 1024, None).expect("creating a new wasm3 runtime should succeed");
        let instance = rt.load_module(module).expect("instance creation should succeed");
        let func = instance.find_function::<(), ()>("call_verification_internal").expect("finding the entrypoint function should succeed");
        b.iter(|| {
            func.call(()).expect("function call should succeed");
        });
    }
}
