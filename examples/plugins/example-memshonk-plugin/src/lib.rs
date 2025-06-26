use core::{ffi::c_void, ptr};

use std::{error::Error, io::Write, sync::OnceLock};

static READ_FROM_PROCESS: OnceLock<ReadFromProcessSig> = OnceLock::new();
type ReadFromProcessSig = extern "C" fn(into: *mut c_void, size: usize, from: usize) -> *mut u8;

#[no_mangle]
extern "C" fn version() -> u16 {
    0
}

#[no_mangle]
extern "C" fn unload() {}

#[no_mangle]
extern "C" fn alloc_v0(size: u32) -> *mut u8 {
    SharedBufOwned::new(size).share()
}

#[no_mangle]
extern "C" fn free_v0(buf: *mut u8) {
    if buf.is_null() {
        return;
    }

    drop(buf.reclaim_vec());
}

#[no_mangle]
extern "C" fn set_read_from_process_v0(func_ptr: ReadFromProcessSig) -> u32 {
    READ_FROM_PROCESS.get_or_init(|| func_ptr);

    0
}

#[no_mangle]
extern "C" fn parsers_v0() -> *mut u8 {
    "example_parser broken_parser".share()
}

use serde::{Deserialize, Serialize};

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
    let data = read_from_process(24, addr)?;

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

fn read_from_process(size: usize, src_adr: usize) -> Result<Vec<u8>, Box<dyn Error>> {
    let func = READ_FROM_PROCESS.get().unwrap();

    let mut dst = vec![0u8; size];

    let err_ptr = func(dst.as_mut_ptr() as *mut c_void, size, src_adr);
    if !err_ptr.is_null() {
        Err(SharedBufRef::reclaim_error(err_ptr))?
    }

    Ok(dst)
}

pub trait ShareableType {
    fn share(self) -> *mut u8;
}

impl ShareableType for &str {
    fn share(self) -> *mut u8 {
        self.to_string().share()
    }
}

impl ShareableType for String {
    fn share(self) -> *mut u8 {
        let mut tmp = SharedBufOwned::new(self.len() as u32);

        write!(tmp.0, "{self}").unwrap();

        tmp.share()
    }
}

impl ShareableType for Box<dyn Error> {
    fn share(self) -> *mut u8 {
        let str = self.to_string();

        str.share()
    }
}

struct SharedBufOwned(Vec<u8>);

impl SharedBufOwned {
    fn new(size: u32) -> Self {
        let mut buf: Vec<u8> = Vec::with_capacity(size as usize + 4);

        _ = buf.write(&u32::to_le_bytes(size)).unwrap();

        SharedBufOwned(buf)
    }
}

impl ShareableType for SharedBufOwned {
    fn share(mut self) -> *mut u8 {
        let ptr = self.0.as_mut_ptr();

        std::mem::forget(self);

        ptr
    }
}

pub trait SharedBufRef {
    fn reclaim_vec(self) -> Vec<u8>;

    fn reclaim_error(self) -> Box<dyn Error>;
}

impl SharedBufRef for *mut u8 {
    fn reclaim_error(self) -> Box<dyn Error> {
        let vec = self.reclaim_vec();

        String::from_utf8_lossy(&vec).into()
    }

    fn reclaim_vec(self) -> Vec<u8> {
        let size_slice = unsafe { std::slice::from_raw_parts(self, 4) };

        let size = u32::from_le_bytes(size_slice.try_into().unwrap()) as usize + 4;

        //let start_of_data = unsafe { self.offset(4) };

        let mut vec: Vec<u8> = unsafe { Vec::from_raw_parts(self, size, size) };

        vec.drain(0..4);

        vec
    }
}
