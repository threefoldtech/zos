use super::hub;
use super::kparams;
use super::workdir::WorkDir;
use super::zfs::Zfs;
use super::zinit;

use failure::Error;
use retry;

const FLIST_REPO: &str = "tf-zos";
const FLIST_INFO_FILE: &str = "/tmp/flist.info";
const FLIST_NAME_FILE: &str = "/tmp/flist.name";
const WORKDIR: &str = "/tmp/bootstrap";

type Result<T> = std::result::Result<T, Error>;

#[derive(Debug)]
enum RunMode {
    Prod,
    Test,
    Dev,
}

fn runmode() -> Result<RunMode> {
    let params = kparams::params()?;
    let mode = match params.get("runmode") {
        Some(mode) => match mode {
            Some(mode) => match mode.as_ref() {
                "prod" => RunMode::Prod,
                "dev" => RunMode::Dev,
                "test" => RunMode::Test,
                m => {
                    bail!("unknown runmode: {}", m);
                }
            },
            None => {
                //that's an error because runmode was
                //provided as a kernel argumet but with no
                //value
                bail!("missing runmode value");
            }
        },
        // runmode was not provided in cmdline
        // so default is prod
        None => RunMode::Prod,
    };

    Ok(mode)
}

fn boostrap_zos(mode: RunMode) -> Result<()> {
    let flist = match mode {
        RunMode::Prod => "zos:production:latest.flist",
        RunMode::Dev => "zos:development:latest.flist",
        RunMode::Test => "zos:testing:latest.flist",
    };

    debug!("using flist: {}/{}", FLIST_REPO, flist);
    let repo = hub::Repo::new(FLIST_REPO);
    let flist = retry::retry(retry::delay::Exponential::from_millis(200), || {
        repo.get(flist)
    });

    let flist = match flist {
        Ok(flist) => flist,
        Err(e) => bail!("failed to download flist: {:?}", e),
    };

    // write down boot info for other system components (like upgraded)
    flist.write(FLIST_INFO_FILE)?;
    std::fs::write(FLIST_NAME_FILE, format!("{}/{}", FLIST_REPO, flist.name))?;

    flist.download("machine.flist")?;

    let fs = Zfs::mount("backend", "machine.flist", "root")?;
    debug!("zfs started, now copying all files");
    fs.copy("/")?; //copy everything to root

    // we need to find all yaml files under /etc/zinit to start monitoring them
    let mut cfg = std::path::PathBuf::new();
    cfg.push(fs);
    cfg.push("etc");
    cfg.push("zinit");
    let services = std::fs::read_dir(&cfg)?;
    for service in services {
        let service = service?;
        let path = service.path();

        if !path.is_file() {
            continue;
        }
        let name = match path.file_name() {
            Some(name) => match name.to_str() {
                Some(name) => name,
                None => {
                    warn!("failed to process name: {:?}", path);
                    continue;
                }
            },
            None => continue,
        };

        match name.rfind(".yaml") {
            None => continue,
            Some(idx) => {
                zinit::monitor(&name[0..idx])?;
            }
        }
    }
    Ok(())
}

pub fn bootstrap() -> Result<()> {
    let mode = runmode()?;
    debug!("runmode: {:?}", mode);
    let result = WorkDir::run(WORKDIR, || -> Result<()> {
        boostrap_zos(mode)?;
        Ok(())
    })?;

    result
}
