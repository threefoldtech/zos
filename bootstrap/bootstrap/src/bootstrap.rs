use failure::Error;

use super::kparams;

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

pub fn bootstrap() -> Result<()> {
    let mode = runmode()?;
    debug!("runmode: {:?}", mode);
    Ok(())
}
