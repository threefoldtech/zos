use storage_primitives::*;

use simplelog::{Config, LevelFilter, TermLogger};

fn main() {
    let _ = TermLogger::init(LevelFilter::Trace, Config::default()).unwrap();

    let mut executor = executor::System::new();
    let _ = fs::FilesystemManager::new(&mut executor);
}
