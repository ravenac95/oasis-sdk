//! Storage imports.
use std::convert::TryInto;

use oasis_contract_sdk_types::storage::StoreKind;
use oasis_runtime_sdk::{context::Context, storage::Store};

use super::{memory::Region, OasisV1};
use crate::{
    abi::{gas, ExecutionContext},
    store, Config, Error,
};

impl<Cfg: Config> OasisV1<Cfg> {
    /// Link storage functions.
    pub fn link_storage<C: Context>(
        instance: &mut wasm3::Instance<'_, '_, ExecutionContext<'_, C>>,
    ) -> Result<(), Error> {
        // storage.get(store, key) -> value
        let _ = instance.link_function(
            "storage",
            "get",
            |ctx, (store, key): (u32, (u32, u32))| -> Result<u32, wasm3::Trap> {
                // Make sure function was called in valid context.
                let ec = ctx.context.ok_or(wasm3::Trap::Abort)?;

                ensure_key_size(ec, key.1)?;

                // Charge base gas amount plus size-dependent gas.
                let total_gas = (|| {
                    let base = ec.params.gas_costs.wasm_storage_get_base;
                    let key = ec
                        .params
                        .gas_costs
                        .wasm_storage_key_byte
                        .checked_mul(key.1.into())?;
                    let total = base.checked_add(key)?;
                    Some(total)
                })()
                .ok_or(wasm3::Trap::Abort)?;
                gas::use_gas(ctx.instance, total_gas)?;

                // Read from contract state.
                let value = ctx.instance.runtime().try_with_memory(
                    |memory| -> Result<_, wasm3::Trap> {
                        let key = Region::from_arg(key).as_slice(&memory)?;
                        Ok(get_instance_store(ec, store)?.get(key))
                    },
                )??;

                let value = match value {
                    Some(value) => value,
                    None => return Ok(0),
                };

                // Charge gas for size of value.
                gas::use_gas(
                    ctx.instance,
                    ec.params
                        .gas_costs
                        .wasm_storage_value_byte
                        .checked_mul(value.len().try_into()?)
                        .ok_or(wasm3::Trap::Abort)?,
                )?;

                // Create new region by calling `allocate`.
                //
                // This makes sure that the call context is unset to avoid any potential issues
                // with reentrancy as attempting to re-enter one of the linked functions will fail.
                let value_region = Self::allocate_and_copy(ctx.instance, &value)?;

                // Return a pointer to the region.
                Self::allocate_region(ctx.instance, value_region).map_err(|e| e.into())
            },
        );

        // storage.insert(store, key, value)
        let _ = instance.link_function(
            "storage",
            "insert",
            |ctx, (store, key, value): (u32, (u32, u32), (u32, u32))| {
                // Make sure function was called in valid context.
                let ec = ctx.context.ok_or(wasm3::Trap::Abort)?;

                ensure_key_size(ec, key.1)?;
                ensure_value_size(ec, value.1)?;

                // Charge base gas amount plus size-dependent gas.
                let total_gas = (|| {
                    let base = ec.params.gas_costs.wasm_storage_insert_base;
                    let key = ec
                        .params
                        .gas_costs
                        .wasm_storage_key_byte
                        .checked_mul(key.1.into())?;
                    let value = ec
                        .params
                        .gas_costs
                        .wasm_storage_value_byte
                        .checked_mul(value.1.into())?;
                    let total = base.checked_add(key)?.checked_add(value)?;
                    Some(total)
                })()
                .ok_or(wasm3::Trap::Abort)?;
                gas::use_gas(ctx.instance, total_gas)?;

                // Insert into contract state.
                ctx.instance
                    .runtime()
                    .try_with_memory(|memory| -> Result<(), wasm3::Trap> {
                        let key = Region::from_arg(key).as_slice(&memory)?;
                        let value = Region::from_arg(value).as_slice(&memory)?;
                        get_instance_store(ec, store)?.insert(key, value);
                        Ok(())
                    })??;

                Ok(())
            },
        );

        // storage.remove(store, key)
        let _ = instance.link_function(
            "storage",
            "remove",
            |ctx, (store, key): (u32, (u32, u32))| {
                // Make sure function was called in valid context.
                let ec = ctx.context.ok_or(wasm3::Trap::Abort)?;

                ensure_key_size(ec, key.1)?;

                // Charge base gas amount plus size-dependent gas.
                let total_gas = (|| {
                    let base = ec.params.gas_costs.wasm_storage_remove_base;
                    let key = ec
                        .params
                        .gas_costs
                        .wasm_storage_key_byte
                        .checked_mul(key.1.into())?;
                    let total = base.checked_add(key)?;
                    Some(total)
                })()
                .ok_or(wasm3::Trap::Abort)?;
                gas::use_gas(ctx.instance, total_gas)?;

                // Remove from contract state.
                ctx.instance
                    .runtime()
                    .try_with_memory(|memory| -> Result<(), wasm3::Trap> {
                        let key = Region::from_arg(key).as_slice(&memory)?;
                        get_instance_store(ec, store)?.remove(key);
                        Ok(())
                    })??;

                Ok(())
            },
        );

        Ok(())
    }
}

/// Create a contract instance store.
fn get_instance_store<'a, C: Context>(
    ec: &'a mut ExecutionContext<'_, C>,
    store_kind: u32,
) -> Result<Box<dyn Store + 'a>, wasm3::Trap> {
    // Determine which store we should be using.
    let store_kind: StoreKind = store_kind.try_into().map_err(|_| wasm3::Trap::Abort)?;

    Ok(store::for_instance(
        ec.tx_context,
        ec.instance_info,
        store_kind,
    )?)
}

/// Make sure that the key size is within the range specified in module parameters.
fn ensure_key_size<C: Context>(ec: &ExecutionContext<'_, C>, size: u32) -> Result<(), wasm3::Trap> {
    if size > ec.params.max_storage_key_size_bytes {
        // TODO: Consider returning a nicer error message.
        return Err(wasm3::Trap::Abort);
    }
    Ok(())
}

/// Make sure that the value size is within the range specified in module parameters.
fn ensure_value_size<C: Context>(
    ec: &ExecutionContext<'_, C>,
    size: u32,
) -> Result<(), wasm3::Trap> {
    if size > ec.params.max_storage_value_size_bytes {
        // TODO: Consider returning a nicer error message.
        return Err(wasm3::Trap::Abort);
    }
    Ok(())
}

#[cfg(test)]
mod test {
    extern crate test;
    use super::*;
    use test::Bencher;

    use oasis_runtime_sdk::{context::Context, keymanager::KeyPair, storage, testing::mock::Mock};

    const BENCH_CODE: &[u8] = include_bytes!("../../../../../../tests/contracts/bench/target/wasm32-unknown-unknown/release/bench.wasm");

    fn make_items(num: usize) -> Vec<(Vec<u8>, Vec<u8>)> {
        let mut items = Vec::new();
        for i in 0..num {
            items.push((
                format!("key{}", i).into_bytes(),
                format!("value{}", i).into_bytes(),
            ));
        }
        items
    }

    #[bench]
    fn bench_wasm_plain_get(b: &mut Bencher) {
        // Set up storage stack and insert some items into it.
        let mut mock = Mock::default();
        let mut ctx = mock.create_ctx();
        let inner = storage::PrefixStore::new(
            storage::PrefixStore::new(
                storage::PrefixStore::new(ctx.runtime_state(), "test module"),
                "instance prefix",
            ),
            "type prefix",
        );
        let mut store = Box::new(storage::HashedStore::<_, blake3::Hasher>::new(inner));

        let items = make_items(10_000);
        for i in 0..10_000 {
            let item = &items[i % items.len()];
            store.insert(&item.0, &item.1);
        }

        // Set up wasm runtime.
        let env = wasm3::Environment::new().expect("creating a new wasm3 environment should succeed");
        let module = env.parse_module(BENCH_CODE).expect("parsing the code should succeed");
        let rt: wasm3::Runtime<'_, wasm3::CallContext<'_, ()>> = env.new_runtime(1 * 1024 * 1024, None).expect("creating a new wasm3 runtime should succeed");
        //let rt = env.new_runtime(1 * 1024 * 1024, None).expect("creating a new wasm3 runtime should succeed");
        let mut instance = rt.load_module(module).expect("instance creation should succeed");
        instance.link_function("bench", "plain_get", move |ctx, key: (u32, u32)| -> Result<(u32, u32), wasm3::Trap> {
            ctx.instance.runtime().try_with_memory(|mut memory| -> Result<_, wasm3::Trap> {
                let key = Region::from_arg(key)
                    .as_slice(&memory)
                    .map_err(|_| wasm3::Trap::Abort)?;
                match store.get(key) {
                    None => Ok((0, 0)),
                    Some(value) => {
                        let alloc = ctx.instance.find_function::<u32, u32>("alloc").expect("finding alloc function should succeed");
                        let target_offset = alloc.call(value.len() as u32).expect("alloc should succeed") as usize;
                        let target = &mut memory.as_slice_mut()[target_offset..target_offset + value.len()];
                        target.copy_from_slice(&value);
                        Ok((target_offset as u32, value.len() as u32))
                    }
                }
            })?
        });
        let func = instance.find_function::<(), ()>("bench_storage").expect("finding the entrypoint function should succeed");
        b.iter(|| {
            func.call(()).expect("function call should succeed");
        });
    }
}
