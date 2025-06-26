use core::ptr;

use mskit::ShareableType;
use serde::{Deserialize, Serialize};
use std::error::Error;

#[no_mangle]
extern "C" fn version() -> u16 {
    0
}

#[no_mangle]
extern "C" fn unload() {}

#[no_mangle]
extern "C" fn parsers_v0() -> *mut u8 {
    "example_parser broken_parser".share()
}

#[derive(Debug, Serialize, Deserialize)]
struct ExampleStruct {
    data: String,
}

#[no_mangle]
extern "C" fn example_parser(addr: usize, str_ptr: *mut *mut u8) -> *mut u8 {
    match _example_parser(addr) {
        Ok(str) => {
            unsafe { *str_ptr = str.share() };

            ptr::null_mut()
        }
        Err(err) => err.share(),
    }
}

fn _example_parser(addr: usize) -> Result<String, Box<dyn Error>> {
    let data = mskit::read_from_process(24, addr)?;

    let tmp = ExampleStruct {
        data: format!("{:#04X?}", data),
    };

    let json = match serde_json::to_string_pretty(&tmp) {
        Ok(r) => r,
        Err(err) => {
            return Err(err)?;
        }
    };

    Ok(json)
}

#[no_mangle]
extern "C" fn broken_parser(_: usize, str_ptr: *mut *mut u8) -> *mut u8 {
    match _broken_parser(str_ptr) {
        Ok(_) => ptr::null_mut(),
        Err(err) => err.share(),
    }
}

fn _broken_parser(_: *mut *mut u8) -> Result<(), Box<dyn Error>> {
    Err("whoops, this is broken")?
}
