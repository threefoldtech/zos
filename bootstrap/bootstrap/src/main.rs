#[macro_use]
extern crate failure;
#[macro_use]
extern crate log;

use clap::{App, Arg};
use failure::Error;

mod bootstrap;
mod hub;
mod kparams;
mod workdir;
mod zfs;
mod zinit;

type Result<T> = std::result::Result<T, Error>;

struct Config {
    stage: u32,
    debug: bool,
}
impl Config {
    fn current() -> Result<Config> {
        let matches = App::new("bootstrap")
            .author("Muhamad Azmy <muhamad.azmy@gmail.com>")
            .about("bootstraps zos from minimal image")
            .arg(
                Arg::with_name("stage")
                    .short("s")
                    .value_name("STAGE")
                    .takes_value(true)
                    .required(false)
                    .default_value("1")
                    .help("specify the bootstrap starting stage"),
            )
            .arg(
                Arg::with_name("debug")
                    .short("d")
                    .takes_value(false)
                    .help("run in debug mode, will use the bootstrap:development.flist"),
            )
            .get_matches();

        let stage: u32 = match matches.value_of("stage").unwrap().parse() {
            Ok(stage) => stage,
            Err(err) => {
                bail!("invalid stage format expecting a positive integer: {}", err);
            }
        };

        if stage == 0 {
            bail!("invalid stage value 0, stages starting from 1");
        }

        Ok(Config {
            stage: stage,
            debug: matches.occurrences_of("debug") > 0,
        })
    }
}

fn app() -> Result<()> {
    let config = Config::current()?;

    let level = if config.debug {
        log::Level::Debug
    } else {
        log::Level::Info
    };

    simple_logger::init_with_level(level).unwrap();

    // configure available stage
    let stages: Vec<fn() -> Result<()>> = vec![
        || -> Result<()> {
            info!("fun stage");
            Ok(())
        },
        || -> Result<()> { bootstrap::bootstrap() },
    ];
    let index = config.stage as usize - 1;

    if index >= stages.len() {
        bail!(
            "unknown stage '{}' only {} stage(s) are supported",
            config.stage,
            stages.len()
        );
    }

    info!("running stage {}/{}", config.stage, stages.len());
    stages[index]()?;

    // Why we run stages in different "processes" (hence using exec)
    // the point is that will allow the stages to do changes to the
    // bootstrap binary. It means an old image with an old version of
    // bootstrap will still be able to run latest code. Because always
    // the first stage is to update self.
    let next = config.stage as usize + 1;
    if next <= stages.len() {
        debug!("spawning stage: {}", next);
        let bin: Vec<String> = std::env::args().take(1).collect();
        let mut cmd = exec::Command::new(&bin[0]);
        let cmd = cmd.arg("-s").arg(format!("{}", next));
        let cmd = if config.debug { cmd.arg("-d") } else { cmd };

        //this call will never return unless something is wrong.
        bail!("{}", cmd.exec());
    }

    Ok(())
}

fn main() {
    let code = match app() {
        Ok(_) => 0,
        Err(err) => {
            eprintln!("{}", err);
            1
        }
    };

    std::process::exit(code);
}
