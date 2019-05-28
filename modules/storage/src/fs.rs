use crate::disks::DiskManager;
use crate::executor;

#[derive(Debug)]
pub struct FilesystemManager<'a> {
    dm: DiskManager<'a>,
}

impl<'a> FilesystemManager<'a> {
    pub fn new(executor: &'a mut dyn executor::Executor) -> Self {
        FilesystemManager {
            dm: DiskManager::new(executor),
        }
    }
}
