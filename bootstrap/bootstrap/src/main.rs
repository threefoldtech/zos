#[macro_use]
extern crate failure;
#[macro_use]
extern crate log;

mod bootstrap;
mod hub;
mod kparams;
mod workdir;
mod zfs;
mod zinit;

fn main() {
    simple_logger::init_with_level(log::Level::Info).unwrap();
    info!("bootstrapping!");

    match bootstrap::bootstrap() {
        Ok(_) => info!("bootstrapping complete"),
        Err(err) => {
            info!("bootstraping failed with err: {}", err);
            std::process::exit(1);
        }
    };

    std::process::exit(0);
}
