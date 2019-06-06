use crate::{disks::DiskManager, executor};

#[derive(Debug)]
pub struct VolumeManager<'a> {
    disk_manager: DiskManager<'a>,
}

impl<'a> VolumeManager<'a> {
    pub fn new(executor: &'a mut dyn executor::Executor) -> Self {
        VolumeManager {
            disk_manager: DiskManager::new(executor),
        }
    }
}
