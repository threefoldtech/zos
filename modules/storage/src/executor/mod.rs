use std::path::Path;

mod system;

pub use system::System;

pub trait Executor {
    fn lsblk(&self) -> Result<String>;
    fn partition_disk(&mut self, node: &crate::disks::Node) -> Result<()>;
    fn create_fs(&mut self, node: &crate::disks::Node) -> Result<()>;
    fn make_fs(&mut self, node: &crate::disks::Node, label: &str) -> Result<()>;
    fn create_btrfs_subvol(&mut self, path: &Path) -> Result<()>;
    fn delete_btrfs_subvol(&mut self, path: &Path) -> Result<()>;
    fn delete_dir(&mut self, path: &Path) -> Result<()>;
    fn list_dir(&mut self, path: &Path) -> Result<Vec<std::fs::DirEntry>>;
    fn mount(&mut self, device: &str, dir: &Path, fs_type: Option<&str>) -> Result<bool>;
    fn copy_dir(&mut self, source: &Path, target: &Path) -> Result<()>;
    fn make_dir(&mut self, path: &Path) -> Result<()>;
    fn is_directory_mountpoint(&self, path: &Path) -> Result<bool>;
    fn btrfs_repair(&mut self, path: &Path) -> Result<bool>;
}

#[derive(Debug)]
pub enum Error {
    IOError(std::io::Error),
    UnknownExitCode,
}

impl std::error::Error for Error {}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        match self {
            Error::IOError(x) => write!(f, "IO Error: {}", x),
            Error::UnknownExitCode => write!(f, "Couldn't determine exit code"),
        }
    }
}

impl From<std::io::Error> for Error {
    fn from(e: std::io::Error) -> Self {
        Error::IOError(e)
    }
}

pub type Result<T> = std::result::Result<T, Error>;
