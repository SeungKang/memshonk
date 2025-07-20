use core::ffi::c_void;

use std::{error::Error, io::Write, sync::OnceLock};

static READ_FROM_PROCESS: OnceLock<ReadFromProcessSig> = OnceLock::new();
type ReadFromProcessSig =
    extern "C" fn(pluginAddr: *mut c_void, size: usize, procAddr: usize) -> *mut u8;

static WRITE_TO_PROCESS: OnceLock<WriteToProcessSig> = OnceLock::new();
type WriteToProcessSig =
    extern "C" fn(procAddr: usize, size: usize, pluginAddr: *mut c_void) -> *mut u8;

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

pub fn read_from_process(size: usize, src_adr: usize) -> Result<Vec<u8>, Box<dyn Error>> {
    let func = READ_FROM_PROCESS.get().unwrap();

    let mut dst = vec![0u8; size];

    let err_ptr = func(dst.as_mut_ptr() as *mut c_void, size, src_adr);
    if !err_ptr.is_null() {
        Err(SharedBufRef::reclaim_error(err_ptr))?
    }

    Ok(dst)
}

#[no_mangle]
extern "C" fn set_write_to_process_v0(func_ptr: WriteToProcessSig) -> u32 {
    WRITE_TO_PROCESS.get_or_init(|| func_ptr);

    0
}

pub fn write_to_process(dst_adr: usize, data: Vec<u8>) -> Result<(), Box<dyn Error>> {
    let func = WRITE_TO_PROCESS.get().unwrap();

    let err_ptr = func(dst_adr, data.len(), data.as_ptr() as *mut c_void);
    if !err_ptr.is_null() {
        Err(SharedBufRef::reclaim_error(err_ptr))?
    }

    Ok(())
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

    fn reclaim_null_string_vec(self) -> Vec<String>;
}

impl SharedBufRef for *mut u8 {
    fn reclaim_error(self) -> Box<dyn Error> {
        let vec = self.reclaim_vec();

        String::from_utf8_lossy(&vec).into()
    }

    fn reclaim_null_string_vec(self) -> Vec<String> {
        let vec = self.reclaim_vec();

        vec.split(|i|*i==0x00)
            .map(|s| String::from_utf8_lossy(s).into_owned())
            .collect()
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
