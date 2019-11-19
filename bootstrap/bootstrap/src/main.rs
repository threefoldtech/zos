#[macro_use]
extern crate failure;
#[macro_use]
extern crate log;

mod bootstrap;
mod hub;
mod kparams;

fn main() {
    simple_logger::init().unwrap();
    info!("bootstrapping!");

    match bootstrap::bootstrap() {
        Ok(_) => info!("bootstrapping complete"),
        Err(err) => {
            info!("bootstraping failed with err: {}", err);
            std::process::exit(1);
        }
    };
}
