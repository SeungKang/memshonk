use core::ptr;

use argparse::Store;
use mskit::{ShareableType, SharedBufRef};
use serde::{Deserialize, Serialize};
use std::error::Error;
use std::str::FromStr;

#[no_mangle]
extern "C" fn version() -> u16 {
    0
}

#[no_mangle]
extern "C" fn description_v0() -> *mut u8 {
    "this is an example memshonk plugin".share()
}

#[no_mangle]
extern "C" fn unload() {}

#[derive(Debug, Serialize, Deserialize)]
struct ExampleStruct {
    data: String,
}

#[no_mangle]
extern "C" fn example_parser_mspar(_: usize, addr: usize, str_ptr: *mut *mut u8) -> *mut u8 {
    match example_parser(addr) {
        Ok(str) => {
            unsafe { *str_ptr = str.share() };

            ptr::null_mut()
        }
        Err(err) => err.share(),
    }
}

fn example_parser(addr: usize) -> Result<String, Box<dyn Error>> {
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
extern "C" fn broken_parser_mspar(_: usize, _: usize, _: *mut *mut u8) -> *mut u8 {
    match broken_parser() {
        Ok(_) => ptr::null_mut(),
        Err(err) => err.share(),
    }
}

fn broken_parser() -> Result<(), Box<dyn Error>> {
    Err("whoops, this is broken")?
}

#[no_mangle]
extern "C" fn example_command_mscmd(_: usize, args: *mut u8, output_ptr: *mut *mut u8) -> *mut u8 {
    match example_command(args.reclaim_null_string_vec()) {
        Ok(str) => {
            unsafe { *output_ptr = str.share() };

            ptr::null_mut()
        }
        Err(err) => err.share(),
    }
}

fn example_command(args_list: Option<Vec<String>>) -> Result<String, Box<dyn Error>> {
    if args_list.is_none() {
        return Err("please specify at least one argument")?;
    }

    let args_list = args_list.unwrap();

    let mut args = ExampleCommandArgs {
        addr: HexAddr(0),
        value: HexValue(Vec::new()),
    };

    let mut parser = argparse::ArgumentParser::new();

    parser
        .refer(&mut args.addr)
        .add_option(&["--addr"], Store, "the address to write to");

    parser
        .refer(&mut args.value)
        .add_option(&["--value"], Store, "the value to write");

    let mut stdout = Vec::new();
    let mut stderr = Vec::new();

    if parser.parse(args_list, &mut stdout, &mut stderr).is_err() {
        if stdout.is_empty() {
            return Err(String::from_utf8(stderr).unwrap())?;
        }

        return Ok(String::from_utf8(stdout)?);
    }

    drop(parser);

    mskit::write_to_process(args.addr.0, args.value.0)?;

    Ok("".into())
}

struct ExampleCommandArgs {
    addr: HexAddr,
    value: HexValue,
}

struct HexAddr(usize);
impl FromStr for HexAddr {
    type Err = Box<dyn Error>;

    fn from_str(mut s: &str) -> Result<Self, Self::Err> {
        s = s.strip_prefix("0x").unwrap_or(s);

        let value = u64::from_str_radix(s, 16)?;
        Ok(HexAddr(value as usize))
    }
}

impl Clone for HexAddr {
    fn clone(&self) -> Self {
        HexAddr(self.0.clone())
    }
}

struct HexValue(Vec<u8>);
impl FromStr for HexValue {
    type Err = Box<dyn Error>;

    fn from_str(mut s: &str) -> Result<Self, Self::Err> {
        s = s.strip_prefix("0x").unwrap_or(s);
        let bytes = if s.len() % 2 != 0 {
            let mut owned = String::new();
            owned.push('0');
            owned.push_str(s);
            hex_to_bytes(&owned)
        } else {
            hex_to_bytes(s)
        };

        Ok(HexValue(bytes.unwrap_or(Vec::new())))
    }
}

impl Clone for HexValue {
    fn clone(&self) -> Self {
        HexValue(self.0.clone())
    }
}

fn hex_to_bytes(s: &str) -> Option<Vec<u8>> {
    (0..s.len())
        .step_by(2)
        .map(|i| {
            s.get(i..i + 2)
                .and_then(|sub| u8::from_str_radix(sub, 16).ok())
        })
        .collect()
}
