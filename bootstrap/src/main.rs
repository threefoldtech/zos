#[macro_use]
extern crate failure;

use failure::Error;
use std::collections::HashMap;
use std::fs;

mod hub;

const FLIST_DEFAULT: &str = "azmy/zos-refs_heads_master.flist";
const HUB_URL: &str = "https://hub.grid.tf/";
const KERNEL_CMD: &str = "/proc/cmdline";
const FILE_VER: &str = "/tmp/version";
const FILE_BOOT: &str = "/tmp/boot";

type Result<T> = std::result::Result<T, Error>;

trait Args {
    fn last(&self, k: &str) -> Option<&str>;
}

impl Args for HashMap<String, Vec<String>> {
    fn last(&self, k: &str) -> Option<&str> {
        let v = match self.get(k) {
            Some(v) => v,
            None => return None,
        };

        if v.len() == 0 {
            return None;
        }

        Some(&v[v.len() - 1])
    }
}

fn args<P>(path: P) -> Result<impl Args>
where
    P: AsRef<std::path::Path>,
{
    let parts = match shlex::split(&fs::read_to_string(path)?) {
        Some(parts) => parts,
        None => bail!("invalid arguments file"),
    };

    let mut result: HashMap<String, Vec<String>> = HashMap::new();

    for part in parts {
        let values: Vec<&str> = part.splitn(2, "=").collect();
        let key = String::from(values[0]);

        if !result.contains_key(&key) {
            result.insert(key.clone(), Vec::new());
        }

        if values.len() == 2 {
            let l = result.get_mut(&key).unwrap();
            l.push(String::from(values[1]));
        }
    }

    Ok(result)
}

fn run() -> Result<()> {
    // get boot flist value
    let kargs = args(KERNEL_CMD)?;
    let flist = kargs.last("flist").unwrap_or(FLIST_DEFAULT);

    // make a note
    fs::write(FILE_BOOT, flist)?;

    let hub = hub::Hub::new();
    let url = hub.url(&flist);

    Ok(())
}

fn main() {
    println!("Hello, world!");
    run().unwrap();
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_args() {
        let mut a: HashMap<String, Vec<String>> = HashMap::new();
        a.insert(
            String::from("name"),
            vec![String::from("john"), String::from("smith")],
        );

        assert_eq!(a.last("age"), None);
        assert_eq!(a.last("name"), Some("smith"));
    }
}
