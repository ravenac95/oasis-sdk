use k256::{self, ecdsa::{self, signature::Verifier as _}};

use core::ptr::read_volatile;

const MESSAGE: &[u8] = include_bytes!("../data/message.txt");
const SIGNATURE: &[u8] = include_bytes!("../data/signature.bin");
const KEY: &[u8] = include_bytes!("../data/key.bin");

#[cfg(target_arch = "wasm32")]
#[global_allocator]
static ALLOC: wee_alloc::WeeAlloc = wee_alloc::WeeAlloc::INIT;

fn verify_signature(message: &[u8], signature: &[u8], key: &[u8]) -> Result<(), ()> {
    let key = k256::EncodedPoint::from_bytes(KEY).map_err(|_| ())?;
    let sig = ecdsa::Signature::from_der(SIGNATURE).map_err(|_| ())?;
    let verifying_key = ecdsa::VerifyingKey::from_encoded_point(&key).map_err(|_| ())?;
    verifying_key.verify(MESSAGE, &sig).map_err(|_| ())?;
    Ok(())
}

#[link(wasm_import_module = "bench")]
extern "C" {
    #[link_name = "verify_signature"]
    fn bench_verify_signature(message_ptr: u32, message_len: u32, signature_ptr: u32, signature_len: u32, key_ptr: u32, key_len: u32);

    #[link_name = "plain_get"]
    fn bench_storage_plain_get(key_ptr: u32, key_length: u32) -> (u32, u32);
}

#[no_mangle]
pub extern "C" fn call_verification_included() {
    unsafe { bench_verify_signature(
        MESSAGE.as_ptr() as u32,
        MESSAGE.len() as u32,
        SIGNATURE.as_ptr() as u32,
        SIGNATURE.len() as u32,
        KEY.as_ptr() as u32,
        KEY.len() as u32,
    ) }
}

#[no_mangle]
pub extern "C" fn call_verification_internal() {
    verify_signature(MESSAGE, SIGNATURE, KEY).unwrap();
}

#[no_mangle]
pub extern "C" fn alloc(length: u32) -> u32 {
    let data: Vec<u8> = Vec::with_capacity(length as usize);
    let data_ptr = data.as_ptr() as usize;
    std::mem::forget(data);
    data_ptr as u32
}

#[no_mangle]
pub extern "C" fn bench_storage() {
    for i in 0..5_000 {
        let key = format!("key{}", i);
        let exp_value = format!("value{}", i);
        let value = unsafe { bench_storage_plain_get(key.as_ptr() as u32, key.len() as u32) };
        let value = unsafe { std::slice::from_raw_parts(value.0 as *const u8, value.1 as usize) };
    }
}

#[no_mangle]
pub extern "C" fn time_waster() {
}
