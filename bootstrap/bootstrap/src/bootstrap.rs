use crate::hub::Flist;
use crate::hub::Kind;
use crate::hub::Repo;

use super::config;
use super::hub;
use super::workdir::WorkDir;
use super::zfs::Zfs;
use super::zinit;
use anyhow::{Context, Result};
use config::{RunMode, Version};
use retry;

const ZOS_REPO: &str = "tf-zos";
const BIN_REPO_V2: &str = "tf-zos-bins";
const BIN_REPO_V3: &str = "tf-zos-v3-bins";

const FLIST_INFO_FILE: &str = "/tmp/flist.info";
const FLIST_NAME_FILE: &str = "/tmp/flist.name";
const FLIST_TAG_FILE: &str = "/tmp/tag.info";
const BOOTSTAP_FLIST: &str = "bootstrap:latest.flist";

const WORKDIR: &str = "/tmp/bootstrap";

fn bootstrap_zos(cfg: &config::Config) -> Result<()> {
    let flist = match &cfg.runmode {
        RunMode::Prod => match &cfg.version {
            Version::V3 => "zos:production-3:latest.flist",
        },
        RunMode::Dev => match &cfg.version {
            Version::V3 => "zos:development-3:latest.flist",
        },
        RunMode::Test => match &cfg.version {
            Version::V3 => "zos:testing-3:latest.flist",
        },
        RunMode::QA => match &cfg.version {
            Version::V3 => "zos:qa-3:latest.flist",
        },
    };

    debug!("using flist: {}/{}", ZOS_REPO, flist);
    let repo = hub::Repo::new(ZOS_REPO);
    let flist = retry::retry(retry::delay::Exponential::from_millis(200), || {
        info!("get flist info: {}", flist);
        let info = match repo.get(flist) {
            Ok(info) => info,
            Err(err) => {
                error!("failed to get info: {}", err);
                bail!("failed to get info: {}", err);
            }
        };

        Ok(info)
    });

    let flist = match flist {
        Ok(flist) => flist,
        Err(e) => bail!("failed to download flist: {:?}", e),
    };

    // write down boot info for other system components (like upgraded)
    flist.write(FLIST_INFO_FILE)?;
    std::fs::write(FLIST_NAME_FILE, format!("{}/{}", ZOS_REPO, flist.name))?;

    install_package(&flist)
}

/// update stage make sure we are running latest
/// version of bootstrap
pub fn update(_cfg: &config::Config) -> Result<()> {
    let result = WorkDir::run(WORKDIR, || {
        update_bootstrap()?;
        Ok(())
    })?;

    result
}

// find the latest bootstrap binary on the hub. and make sure
// it's installed and available on the system, before starting
// the actual system bootstrap.
fn update_bootstrap() -> Result<()> {
    // we are running in a tmpfs workdir in this method
    let repo = hub::Repo::new("tf-autobuilder");
    let name = BOOTSTAP_FLIST;

    let flist = retry::retry(retry::delay::Exponential::from_millis(200), || {
        info!("get flist info: {}", name);
        //the full point of this match is the logging.
        let info = match repo.get(name) {
            Ok(info) => info,
            Err(err) => {
                error!("failed to get info: {}", err);
                bail!("failed to get info: {}", err);
            }
        };

        Ok(info)
    });

    let flist = match flist {
        Ok(flist) => flist,
        Err(e) => bail!("failed to download flist: {:?}", e),
    };

    // this trick here to allow overriding
    // the current running bootstrap binary
    let bin: Vec<String> = std::env::args().take(1).collect();
    std::fs::rename(&bin[0], format!("{}.bak", &bin[0]))?;

    install_package(&flist)
}

///install installs all binaries from the tf-zos-bins repo
pub fn install(cfg: &config::Config) -> Result<()> {
    let repo = Repo::new(ZOS_REPO);

    let runmode = cfg.runmode.to_string();
    // we need to list all taglinks inside the repo

    let mut tag = None;
    for list in repo.list()? {
        if list.kind == Kind::TagLink && list.name == runmode {
            tag = Some(list);
            break;
        }
    }

    if let Some(ref tag) = tag {
        info!("found tag {} => {:?}", tag.name, tag.target);
    }

    let result = WorkDir::run(WORKDIR, || -> Result<()> {
        match tag {
            None => {
                // old style bootstrap.
                // we need to install binaries and zos from 2 different
                // places
                // we also track which binaries are installed individually
                install_packages_old(cfg)?;
                bootstrap_zos(cfg)
            }
            Some(tag) => {
                // new style bootstrap
                // we need then to
                tag.write(FLIST_TAG_FILE)?;

                let (repo, tag) = tag.tag_link();
                let client = Repo::new(repo);
                let packages = client.list_tag(tag)?.context("tag is not found")?;

                // new style setup, just install every thing.
                install_packages(&packages)

                //TODO: write down which version of the tag is installed
            }
        }
    })?;

    result
}

fn install_packages_old(cfg: &config::Config) -> Result<()> {
    let name = match cfg.version {
        Version::V3 => BIN_REPO_V3,
    };

    let repo = match cfg.runmode {
        config::RunMode::Prod => name.into(),
        config::RunMode::Dev => format!("{}.dev", name),
        config::RunMode::Test => format!("{}.test", name),
        config::RunMode::QA => format!("{}.qanet", name),
    };

    let client = hub::Repo::new(&repo);
    let packages = retry::retry(retry::delay::Exponential::from_millis(200), || {
        info!("list packages in: {}", BIN_REPO_V2);
        //the full point of this match is the logging.
        let packages = match client.list() {
            Ok(info) => info,
            Err(err) => {
                error!("failed to list repo '{}': {}", BIN_REPO_V2, err);
                bail!("failed to list repo '{}': {}", BIN_REPO_V2, err);
            }
        };

        Ok(packages)
    });

    let packages = match packages {
        Ok(packages) => packages,
        Err(err) => bail!("failed to list '{}': {:?}", BIN_REPO_V2, err),
    };

    let mut map = std::collections::HashMap::new();
    for package in packages.iter() {
        match install_package(package) {
            Ok(_) => {}
            Err(err) => warn!("failed to install package '{}': {}", package.url, err),
        };

        map.insert(format!("{}/{}", repo, package.name), package.clone());
    }

    let output = std::fs::OpenOptions::new()
        .create(true)
        .write(true)
        .open("/tmp/bins.info")?;
    serde_json::to_writer(&output, &map)?;

    Ok(())
}

fn install_packages(packages: &[Flist]) -> Result<()> {
    for package in packages {
        install_package(&package)?;
    }

    Ok(())
}

fn install_package(flist: &hub::Flist) -> Result<()> {
    let result = retry::retry(retry::delay::Exponential::from_millis(200), || {
        info!("download flist: {}", flist.name);

        // the entire point of this match is the
        // logging of the error.
        flist
            .download(&flist.name)
            .with_context(|| format!("failed to download flist: {}", flist.url))
    });

    match result {
        Err(retry::Error::Operation { error, .. }) => return Err(error),
        Err(retry::Error::Internal(msg)) => bail!("failed retrying to download flist: {}", msg),
        _ => (),
    };

    let fs = retry::retry(retry::delay::Exponential::from_millis(500).take(10), || {
        Zfs::mount("backend", &flist.name, "root")
            .with_context(|| format!("failed to mount flist: {}", flist.url))
    });

    let fs = match fs {
        Ok(fs) => fs,
        Err(retry::Error::Operation { error, .. }) => return Err(error),
        Err(retry::Error::Internal(msg)) => bail!("failed retrying to mount flist: {}", msg),
    };

    debug!("zfs started, now copying all files");

    fs.copy("/").context("failed to copy files")?;

    debug!("starting services");
    run_all(&fs)
}

// run_all tries to run all services from an flist.
// it will still try to run all other services defined
// in the list of one or more failed. Returns error only
// if failed to read the zinit directory
fn run_all(fs: &Zfs) -> Result<()> {
    let mut cfg = std::path::PathBuf::new();
    cfg.push(fs);
    cfg.push("etc");
    cfg.push("zinit");
    let services = match std::fs::read_dir(&cfg) {
        Ok(services) => services,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => return Ok(()),
        Err(err) => bail!("failed to read directory '{:?}': {}", cfg, err),
    };

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
                let service = &name[0..idx];
                match zinit::monitor(service) {
                    Ok(_) => {}
                    Err(err) => {
                        warn!("failed to monitor service '{}': {}", service, err);
                    }
                }
            }
        }
    }

    Ok(())
}
