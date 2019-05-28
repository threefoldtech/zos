use std::path::Path;
use std::process::Command;

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

        Ok(String::from_utf8(cmd.output()?.stdout).expect("Could not parse lsblk output"))
    }

    fn partition_disk(&mut self, node: &crate::disks::Node) -> super::Result<()> {
        // parted -s {node.path} mktable gpt
        let mut cmd = Command::new("parted");
        cmd.arg("-s");
        cmd.arg(&node.path);
        cmd.arg("mktable");
        cmd.arg("gpt");

        cmd.spawn()?.wait()?;

        Ok(())
    }

    fn create_fs(&mut self, node: &crate::disks::Node) -> super::Result<()> {
        // parted -s {node.path} mkpart primary btrfs 1 100%
        let mut cmd = Command::new("parted");
        cmd.arg("-s");
        cmd.arg(&node.path);
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
        cmd.arg(&node.path);
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
        // rm -rf {dir}
        let mut cmd = Command::new("rm");
        cmd.arg("-rf");
        cmd.arg(path);

        cmd.spawn()?.wait()?;

        Ok(())
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

    fn mount(&mut self, device: &str, dir: &Path, fs_type: Option<&str>) -> super::Result<bool> {
        // mount (-t {fs_type}) {device} {dir}
        let mut cmd = Command::new("mount");
        if let Some(fs_type) = fs_type {
            cmd.arg("-t");
            cmd.arg(fs_type);
        }
        cmd.arg(device);
        cmd.arg(dir);

        match cmd.status()?.code() {
            Some(code) => match code {
                0 => Ok(true),
                _ => Ok(false),
            },
            None => Err(super::Error::UnknownExitCode),
        }
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
        // mkdir -p {path}
        let mut cmd = Command::new("mkdir");
        cmd.arg("-p");
        cmd.arg(path);

        cmd.spawn()?.wait()?;

        Ok(())
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
