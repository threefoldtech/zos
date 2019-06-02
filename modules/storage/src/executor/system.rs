use std::fs;
use std::path::Path;
use std::process::Command;

use nix::mount::{mount, MsFlags};

#[derive(Debug)]
pub struct System;

impl System {
    pub fn new() -> System {
        Self
    }
}

impl super::Executor for System {
    fn lsblk(&self) -> super::Result<String> {
        let mut cmd = Command::new("lsblk");
        cmd.arg("--json"); // json output
        cmd.arg("--bytes");
        cmd.arg("--output-all");
        cmd.arg("--paths"); // use device patch as name

        Ok(String::from_utf8(cmd.output()?.stdout).expect("Could not parse lsblk output"))
    }

    fn partition_disk(&mut self, node: &crate::disks::Node) -> super::Result<()> {
        // parted -s {node.path} mktable gpt
        let mut cmd = Command::new("parted");
        cmd.arg("-s");
        cmd.arg(&node.name);
        cmd.arg("mktable");
        cmd.arg("gpt");

        cmd.spawn()?.wait()?;

        Ok(())
    }

    fn create_fs(&mut self, node: &crate::disks::Node) -> super::Result<()> {
        // parted -s {node.path} mkpart primary btrfs 1 100%
        let mut cmd = Command::new("parted");
        cmd.arg("-s");
        cmd.arg(&node.name);
        cmd.arg("mkpart");
        cmd.arg("primary"); // TODO: make dynamic
        cmd.arg("btrfs"); // TODO: multiple fs types
        cmd.arg("1");
        cmd.arg("100%");

        cmd.spawn()?.wait()?;

        Ok(())
    }

    fn make_fs(&mut self, node: &crate::disks::Node, label: &str) -> super::Result<()> {
        // mkfs.{fs} {node.path} -f -L ${LABEL}
        let mut cmd = Command::new(format!("mkfs.{}", "btrfs"));
        cmd.arg(&node.name);
        cmd.arg("-f");
        cmd.arg("-L");
        cmd.arg(label);

        cmd.spawn()?.wait()?;

        Ok(())
    }

    fn create_btrfs_subvol(&mut self, path: &Path) -> super::Result<()> {
        // btrfs subvol create {path}
        let mut cmd = Command::new("btrfs");
        cmd.arg("subvol");
        cmd.arg("create");
        cmd.arg(path);

        cmd.spawn()?.wait()?;

        Ok(())
    }

    fn delete_btrfs_subvol(&mut self, path: &Path) -> super::Result<()> {
        // btrfs subvol del {path}
        let mut cmd = Command::new("btrfs");
        cmd.arg("subvol");
        cmd.arg("delete");
        cmd.arg(path);

        cmd.spawn()?.wait()?;

        Ok(())
    }

    fn delete_dir(&mut self, path: &Path) -> super::Result<()> {
        Ok(fs::remove_dir_all(path)?)
    }

    fn list_dir(&mut self, path: &Path) -> super::Result<Vec<std::fs::DirEntry>> {
        // we can't use iterator adapters to collect the entries since collect returns a Vec<T>,
        // not a Result<Vec<T>, E>, and the iterator returned by read_dir iterates over Result
        // types
        let mut results = Vec::new();

        for entry in std::fs::read_dir(path)? {
            results.push(entry?);
        }

        Ok(results)
    }

    fn mount(&mut self, device: &Path, dir: &Path, fs_type: Option<&str>) -> super::Result<()> {
        let mut flags = MsFlags::MS_SILENT;
        if device.is_dir() {
            trace!(
                "Mount device ({}) is directory, creating bind mount",
                device.display()
            );
            flags |= MsFlags::MS_BIND;
        }
        trace!("Filesystem type {:?}", fs_type);
        trace!("Mount flags {:?}", flags);
        trace!("mount {} -> {}", device.display(), dir.display());
        Ok(mount::<Path, Path, str, Path>(
            Some(&device),
            dir,
            fs_type,
            flags,
            None,
        )?)
    }

    fn copy_dir(&mut self, source: &Path, target: &Path) -> super::Result<()> {
        // cp -a {source} {dir}
        let mut cmd = Command::new("cp");
        cmd.arg("-a");
        cmd.arg(source);
        cmd.arg(target);

        cmd.spawn()?.wait()?;

        Ok(())
    }

    fn make_dir(&mut self, path: &Path) -> super::Result<()> {
        Ok(fs::create_dir_all(path)?)
    }

    fn is_directory_mountpoint(&self, path: &Path) -> super::Result<bool> {
        // mountpoint -q {path}
        let mut cmd = Command::new("mountpoint");
        cmd.arg("-q");
        cmd.arg(path);

        match cmd.status()?.code() {
            Some(code) => match code {
                0 => Ok(true),
                _ => Ok(false),
            },
            None => Err(super::Error::UnknownExitCode),
        }
    }

    fn btrfs_repair(&mut self, path: &Path) -> super::Result<bool> {
        // btrfs check --repair {path}
        let mut cmd = Command::new("btrfs");
        cmd.arg("check");
        cmd.arg("--repair");
        cmd.arg(path);

        match cmd.status()?.code() {
            Some(code) => match code {
                0 => Ok(true),
                _ => Ok(false),
            },
            None => Err(super::Error::UnknownExitCode),
        }
    }
}
