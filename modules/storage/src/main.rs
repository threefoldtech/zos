use storage_primitives::*;

fn main() {
    let mut executor = executor::System::new();
    let _ = fs::FilesystemManager::new(&mut executor);
}
