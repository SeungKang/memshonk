use core::ptr;
use mskit::{write_to_process, Ctx, ShareableType, SharedBufRef};
use serde::{Deserialize, Serialize};
use std::error::Error;
use std::num::Wrapping;
use std::time::Duration;

#[no_mangle]
extern "C" fn version() -> u32 {
    0x00_00_01_00
}

#[no_mangle]
extern "C" fn unload() {}

// TODO support for hex encoded integer https://stackoverflow.com/questions/67145666/make-serde-only-produce-hex-strings-for-human-readable-serialiser/67148611#67148611
#[derive(Debug, Serialize, Deserialize)]
struct EntityStruct {
    //stuff_before: String,
    struct_addr: usize,
    entity_coords: EntityCoords,
    rot_x_coord: f32,
    rot_y_coord: f32,
    rot_z_coord: f32,
    ground_speed: f32,
    air_speed: f32,
    accel_rate: f32,
    walk_speed: f32,
    combat_walk_speed: f32,
    combat_ground_speed: f32,
    cover_ground_speed: f32,
    low_cover_ground_speed: f32,
    tight_aim_ground_speed: f32,
    crouch_ground_speed: f32,
    storm_speed: f32,
    //stuff_after: String,
}

#[no_mangle]
extern "C" fn parse_geth_troopers_mspar(_: usize, _: usize, str_ptr: *mut *mut u8) -> *mut u8 {
    match parse_geth_troppers() {
        Ok(str) => {
            unsafe { *str_ptr = str.share() };

            ptr::null_mut()
        }
        Err(err) => err.share(),
    }
}

fn parse_geth_troppers() -> Result<String, Box<dyn Error>> {
    let enemies = get_geth_troppers()?;

    let json = serde_json::to_string_pretty(&enemies)?;

    Ok(json)
}

#[no_mangle]
extern "C" fn parse_geth_coords_mspar(_: usize, _: usize, str_ptr: *mut *mut u8) -> *mut u8 {
    match parse_geth_coords() {
        Ok(str) => {
            unsafe { *str_ptr = str.share() };

            ptr::null_mut()
        }
        Err(err) => err.share(),
    }
}

fn parse_geth_coords() -> Result<String, Box<dyn Error>> {
    let enemies = get_geth_troppers()?;
    let enemy_coords: Vec<_> = enemies.iter().map(|e| &e.entity_coords).collect();

    let json = serde_json::to_string_pretty(&enemy_coords)?;

    Ok(json)
}

unsafe fn read_ptr(from: usize) -> Result<usize, Box<dyn Error>> {
    let buf = mskit::read_from_process(4, from)?;
    let buf: [u8; 4] = buf.as_slice()[0..4].try_into()?;

    Ok(u32::from_le_bytes(buf) as usize)
}

#[no_mangle]
extern "C" fn parse_enemy_mspar(_: usize, addr: usize, str_ptr: *mut *mut u8) -> *mut u8 {
    match parse_enemy(addr) {
        Ok(str) => {
            unsafe { *str_ptr = str.share() };

            ptr::null_mut()
        }
        Err(err) => err.share(),
    }
}

fn parse_enemy(addr: usize) -> Result<String, Box<dyn Error>> {
    let enemy = get_entity(addr)?;

    let json = serde_json::to_string_pretty(&enemy)?;

    Ok(json)
}

fn get_entity(addr: usize) -> Result<EntityStruct, Box<dyn Error>> {
    let size = 0x1000;

    let buf = mskit::read_from_process(size, addr)?;

    Ok(EntityStruct {
        //stuff_before: format!("{:x?}", buf[0x0..0xb7].bytes()),
        struct_addr: addr,
        entity_coords: EntityCoords {
            x_coord: float_f32_from_bytes(&buf, 0xb8)?,
            y_coord: float_f32_from_bytes(&buf, 0xbc)?,
            z_coord: float_f32_from_bytes(&buf, 0xc0)?,
        },
        rot_x_coord: float_f32_from_bytes(&buf, 0xc4)?,
        rot_y_coord: float_f32_from_bytes(&buf, 0xc8)?,
        rot_z_coord: float_f32_from_bytes(&buf, 0xcc)?,
        ground_speed: float_f32_from_bytes(&buf, 0x36c)?,
        air_speed: float_f32_from_bytes(&buf, 0x374)?,
        accel_rate: float_f32_from_bytes(&buf, 0x37c)?,
        walk_speed: float_f32_from_bytes(&buf, 0x970)?,
        combat_walk_speed: float_f32_from_bytes(&buf, 0x974)?,
        combat_ground_speed: float_f32_from_bytes(&buf, 0x978)?,
        cover_ground_speed: float_f32_from_bytes(&buf, 0x97c)?,
        low_cover_ground_speed: float_f32_from_bytes(&buf, 0x980)?,
        tight_aim_ground_speed: float_f32_from_bytes(&buf, 0x984)?,
        crouch_ground_speed: float_f32_from_bytes(&buf, 0x988)?,
        storm_speed: float_f32_from_bytes(&buf, 0x98c)?,
        //stuff_after: format!("{:x?}", buf[0xc4..].bytes()),
    })
}

#[derive(Debug, Serialize, Deserialize)]
struct EntityCoords {
    x_coord: f32,
    y_coord: f32,
    z_coord: f32,
}

fn float_f32_from_bytes(buf: &Vec<u8>, index: usize) -> Result<f32, Box<dyn Error>> {
    let x: [u8; 4] = buf[index..index + 4]
        .try_into()
        .map_err(|err| format!("failed to extract buf bytes at index: {index} - {err}"))?;

    Ok(f32::from_le_bytes(x))
}

fn get_geth_troppers() -> Result<Vec<EntityStruct>, Box<dyn Error>> {
    let geth_trooper_start = 0x0197AD28;
    let mut enemies: Vec<EntityStruct> = Vec::new();

    // TODO: this is hardcoded number, should be changed
    for i in 0..30 {
        let target = geth_trooper_start - i * 0x4;
        let enemy_ptr = unsafe { read_ptr(target)? };
        if enemy_ptr == 0 {
            continue;
        }

        let buf = mskit::read_from_process(4, enemy_ptr)?;

        // enemy struct not found
        if buf != vec![0xd8, 0xb, 0x81, 0x1] {
            // eprintln!("enemy_{i} was not found");
            continue;
        }

        let enemy = get_entity(enemy_ptr)?;
        enemies.push(enemy);
    }

    Ok(enemies)
}

#[no_mangle]
extern "C" fn geth_spin_mscmd(ctx: *mut Ctx, args: *mut u8, output_ptr: *mut *mut u8) -> *mut u8 {
    match geth_spin(Ctx::from_ptr(ctx), args.reclaim_null_string_vec()) {
        Ok(str) => {
            unsafe { *output_ptr = str.share() };

            ptr::null_mut()
        }
        Err(err) => err.share(),
    }
}

fn geth_spin(ctx: mskit::Ctx, _: Option<Vec<String>>) -> Result<String, Box<dyn Error>> {
    let mut rotation: Wrapping<f32> = Wrapping(0.0);

    loop {
        rotation.0 += 1000.0;
        let enemies = get_geth_troppers()?;

        let mut gap: f32 = 1.0;
        for enemy in enemies.iter() {
            let enemy_x_coord = -1700.0 + 400.0 * gap;
            let enemy_y_coord: f32 = 3000.0;
            let enemy_z_coord: f32 = 800.0;
            gap += 1.0;

            // eprintln!("enemy.struct_addr: {:#x?}", enemy.struct_addr);
            // x coord
            write_to_process(
                enemy.struct_addr + 0xb8,
                enemy_x_coord.to_le_bytes().to_vec(),
            )?;
            // y coord
            write_to_process(
                enemy.struct_addr + 0xbc,
                enemy_y_coord.to_le_bytes().to_vec(),
            )?;
            // z coord
            write_to_process(
                enemy.struct_addr + 0xc0,
                enemy_z_coord.to_le_bytes().to_vec(),
            )?;

            // x rotation
            write_to_process(enemy.struct_addr + 0xc8, rotation.0.to_le_bytes().to_vec())?;
        }

        if ctx.is_cancelled_timeout(Duration::from_millis(10)) {
            return Ok("".into());
        }
    }
}

#[no_mangle]
extern "C" fn geth_up_mscmd(ctx: *mut Ctx, args: *mut u8, output_ptr: *mut *mut u8) -> *mut u8 {
    match geth_up(Ctx::from_ptr(ctx), args.reclaim_null_string_vec()) {
        Ok(str) => {
            unsafe { *output_ptr = str.share() };

            ptr::null_mut()
        }
        Err(err) => err.share(),
    }
}

fn geth_up(ctx: mskit::Ctx, _: Option<Vec<String>>) -> Result<String, Box<dyn Error>> {
    let enemies = get_geth_troppers()?;
    
    for enemy in enemies.iter() {
        // z coord
        write_to_process(
            enemy.struct_addr + 0xc0,
            (enemy.entity_coords.z_coord + 200.0).to_le_bytes().to_vec(),
        )?;
        
    }

    Ok("".into())
}

#[no_mangle]
extern "C" fn geth_down_mscmd(ctx: *mut Ctx, args: *mut u8, output_ptr: *mut *mut u8) -> *mut u8 {
    match geth_down(Ctx::from_ptr(ctx), args.reclaim_null_string_vec()) {
        Ok(str) => {
            unsafe { *output_ptr = str.share() };

            ptr::null_mut()
        }
        Err(err) => err.share(),
    }
}

fn geth_down(ctx: mskit::Ctx, _: Option<Vec<String>>) -> Result<String, Box<dyn Error>> {
    let enemies = get_geth_troppers()?;

    for enemy in enemies.iter() {
        // z coord
        write_to_process(
            enemy.struct_addr + 0xc0,
            (enemy.entity_coords.z_coord - 200.0).to_le_bytes().to_vec(),
        )?;

    }

    Ok("".into())
}

fn get_player() -> Result<EntityStruct, Box<dyn Error>> {
    // TODO: the address below is not a reliable address to get the player addr
    let player_addr = 0x01979944;
    let player_ptr = unsafe { read_ptr(player_addr)? };
    if player_ptr == 0 {
        Err(format!("player ptr points to null: {:#x?}", player_addr))?;
    }

    let buf = mskit::read_from_process(4, player_ptr)?;

    // entity struct not found
    if buf != vec![0xd8, 0xb, 0x81, 0x1] {
        Err(format!(
            "player struct was not found at: {:#x?}",
            player_ptr
        ))?;
    }

    let player = get_entity(player_ptr)?;

    Ok(player)
}
