use failure::Error;
use fs_extra::dir;
use std::path::{Path, PathBuf};
use std::process::{Child, Command};

type Result<T> = std::result::Result<T, Error>;

pub struct Zfs {
    target: PathBuf,
    child: Child,
}

impl Zfs {
    pub fn mount<S, P>(backend: S, meta: S, target: P) -> Result<Zfs>
    where
        S: AsRef<str>,
        P: Into<PathBuf>,
    {
        // mount the flist
        let target = target.into();
        let mut child = Command::new("g8ufs")
            .arg("--ro")
            .arg("--backend")
            .arg(backend.as_ref())
            .arg("--meta")
            .arg(meta.as_ref())
            .arg(&target)
            .spawn()?;

        //wait for the mount
        let result = retry::retry(retry::delay::Exponential::from_millis(200).take(10), || {
            let status = Command::new("mountpoint").arg("-q").arg(&target).status()?;
            match status.success() {
                true => Ok(()),
                false => bail!("not a mount point"),
            }
        });

        match result {
            Err(e) => {
                child.kill()?;
                bail!("failed to mount flist: {:?}", e)
            }
            Ok(_) => {}
        }

        Ok(Zfs {
            target: target,
            child: child,
        })
    }

    pub fn copy<P>(&self, target: P) -> Result<()>
    where
        P: AsRef<Path>,
    {
        let mut opts = dir::CopyOptions::new();
        opts.overwrite = true;
        opts.copy_inside = true;
        debug!(
            "copying from {:?} -to-> {:?}",
            &self.target,
            target.as_ref()
        );
        /*
         |info: dir::TransitProcess| {
            print!(
                "Copying {} {}/{}\r",
                info.file_name, info.file_bytes_copied, info.file_total_bytes
            );
            dir::TransitProcessResult::ContinueOrAbort
        }
         */
        dir::copy(&self.target, target, &opts)?;

        Ok(())
    }
}

impl AsRef<Path> for Zfs {
    fn as_ref(&self) -> &Path {
        &self.target
    }
}

impl Drop for Zfs {
    fn drop(&mut self) {
        if let Err(e) = nix::mount::umount2(&self.target, nix::mount::MntFlags::MNT_FORCE) {
            error!("failed to umount flist: {}", e);
        }
        if let Err(e) = self.child.wait() {
            error!("g8ufs wait error: {}", e);
        }
    }
}
