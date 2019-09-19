const HUB_URL: &str = "https://hub.grid.tf/";
const HUB_API_URL: &str = "https://hub.grid.tf/api/flist/";

use failure::Error;
use reqwest::{Client, Url};
use semver::Version;
use serde::Deserialize;

type Result<T> = std::result::Result<T, Error>;

#[derive(Deserialize)]
pub struct Info {
    pub md5: String,
    pub name: String,
    pub size: u64,
    #[serde(rename = "type")] // because type is a keyword
    pub kind: String,
    pub target: String,
    pub updated: u64,
}

impl Info {
    pub fn version(&self) -> Option<Version> {
        match self.kind.as_ref() {
            "symlink" => self.extract(&self.target),
            _ => self.extract(&self.name),
        }
    }

    fn extract(&self, v: &str) -> Option<Version> {
        let halves: Vec<&str> = v.splitn(2, ":").collect();
        if halves.len() != 2 {
            return None;
        }

        match halves[1].trim_right_matches(".flist").parse() {
            Ok(version) => Some(version),
            Err(err) => None,
        }
    }
}

pub struct Hub {
    client: Client,
}

impl Hub {
    pub fn new() -> Hub {
        Hub {
            client: Client::new(),
        }
    }

    pub fn url(&self, name: &str) -> Result<String> {
        let u = Url::parse(HUB_URL)?;
        let u = u.join(name)?;

        return Ok(u.to_string());
    }

    pub fn info(&self, name: &str) -> Result<Info> {
        let u = Url::parse(HUB_API_URL)?;
        let u = u.join(name)?;

        Ok(self.client.head(u).send()?.json()?)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn test_hub() {
        let hub = Hub::new();

        let info = hub.info("tf-bootable/ubuntu:18.04.flist").unwrap();

        assert_eq!(info.version(), Some(Version::parse("18.04").unwrap()))
    }

    #[test]
    fn test_info_unknown() {
        let info = Info {
            md5: String::from("xxx"),
            name: String::from("unversioned.flist"),
            target: String::from(""),
            kind: String::from("regular"),
            size: 0,
            updated: 0,
        };

        assert_eq!(info.version(), None);
    }

    #[test]
    fn test_info_regular() {
        let info = Info {
            md5: String::from("xxx"),
            name: String::from("versioned:1.2.3.flist"),
            target: String::from(""),
            kind: String::from("regular"),
            size: 0,
            updated: 0,
        };

        assert_eq!(info.version(), Some(Version::parse("1.2.3").unwrap()));
    }

    #[test]
    fn test_info_symlink() {
        let info = Info {
            md5: String::from("xxx"),
            name: String::from("versioned:latest.flist"),
            target: String::from("versioned:1.2.3.flist"),
            kind: String::from("symlink"),
            size: 0,
            updated: 0,
        };

        assert_eq!(info.version(), Some(Version::parse("1.2.3").unwrap()));
    }
}
