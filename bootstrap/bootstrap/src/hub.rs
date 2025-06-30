use crate::config::RunMode;
use anyhow::Result;
use reqwest::{blocking::get, StatusCode};
use retry::delay::Exponential;
use retry::{retry, OperationResult};
use serde::{Deserialize, Serialize};
use serde_json;
use std::fs::{write, OpenOptions};
use std::io::copy;
use std::path::Path;
use url::Url;

#[derive(Deserialize)]
struct ZosConfig {
    flist_url: Vec<String>,
}

fn get_hub_url(runmode: &RunMode) -> Result<Vec<String>> {
    let base_url = "https://github.com/threefoldtech/zos-config/raw/main/";
    let config_filename = match runmode {
        RunMode::Prod => "production.json",
        RunMode::Dev => "development.json",
        RunMode::Test => "testing.json",
        RunMode::QA => "qa.json",
    };

    let config_url = format!("{}/{}", base_url, config_filename);
    let fallback = vec!["https://hub.grid.tf".to_string()];

    let final_url = retry(Exponential::from_millis(1000).take(5), || {
        match get(config_url.as_str()) {
            Ok(resp) if resp.status().is_success() => OperationResult::Ok(resp),
            Ok(_) | Err(_) => OperationResult::Retry("Retrying..."),
        }
    });
    let response = match final_url {
        Ok(resp) => resp,
        Err(_) => return Ok(fallback),
    };

    let config: ZosConfig = match response.json() {
        Ok(config) => config,
        Err(_) => return Ok(fallback),
    };

    let mut hub_urls = Vec::new();

    for flist_url in config.flist_url {
        let hub_url = if let Ok(parsed) = Url::parse(&flist_url) {
            if let Some(host) = parsed.host_str() {
                format!("https://{}", host)
            } else {
                continue;
            }
        } else {
            continue;
        };

        hub_urls.push(hub_url);
    }

    if hub_urls.is_empty() {
        Ok(fallback)
    } else {
        Ok(hub_urls)
    }
}

pub struct Repo {
    name: String,
    hub: Vec<String>,
}

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq)]
pub enum Kind {
    #[serde(rename = "regular")]
    Regular,
    #[serde(rename = "symlink")]
    Symlink,
    #[serde(rename = "tag")]
    Tag,
    #[serde(rename = "taglink")]
    TagLink,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Flist {
    #[serde(rename = "type")]
    pub kind: Kind,
    pub updated: u64,
    #[serde(default)]
    pub md5: String,
    pub name: String,
    pub target: Option<String>,
    #[serde(skip)]
    pub url: String,
}

impl Repo {
    pub fn new<T>(name: T) -> Result<Repo>
    where
        T: AsRef<str>,
    {
        let config = crate::config::Config::current()?;
        let hub = get_hub_url(&config.runmode)?;
        Ok(Repo {
            name: String::from(name.as_ref()),
            hub,
        })
    }

    /// Helper function to find the first working hub URL
    fn get_working_hub(&self) -> &String {
        for hub_url in &self.hub {
            // Try to ping the hub with a simple API call
            if self.is_hub_working(hub_url) {
                return hub_url;
            }
        }

        &self.hub[0]
    }

    /// Check if a specific hub URL is working
    fn is_hub_working(&self, hub_url: &str) -> bool {
        let health_url = format!("{}/api/flist", hub_url);

        match get(&health_url) {
            Ok(response) => response.status().is_success(),
            Err(_) => false,
        }
    }

    pub fn list(&self) -> Result<Vec<Flist>> {
        let url = format!("{}/api/flist/{}", self.get_working_hub(), self.name,);

        let response = get(&url)?;
        let mut info: Vec<Flist> = match response.status() {
            StatusCode::OK => response.json()?,
            s => bail!("failed to get flist info: {}", s),
        };
        for flist in info.iter_mut() {
            flist.url = format!("{}/{}/{}", self.get_working_hub(), self.name, flist.name);
        }

        Ok(info)
    }

    pub fn list_tag<S: AsRef<str>>(&self, tag: S) -> Result<Option<Vec<Flist>>> {
        let tag = tag.as_ref();

        let url = format!(
            "{}/api/flist/{}/tags/{}",
            self.get_working_hub(),
            self.name,
            tag
        );
        let response = get(&url)?;
        let mut info: Vec<Flist> = match response.status() {
            StatusCode::OK => response.json()?,
            StatusCode::NOT_FOUND => return Ok(None),
            s => bail!("failed to get flist info: {}", s),
        };

        // when listing tags. The flists inside have target as full name
        // so we need
        for flist in info.iter_mut() {
            if flist.kind == Kind::Regular {
                // this is impossible because tags can only have symlinks
                continue;
            }

            let target = match &flist.target {
                None => {
                    // this is also not possible may be we should return an error
                    // but we can just skip for now
                    continue;
                }
                Some(target) => target,
            };

            flist.url = format!("{}/{}", self.get_working_hub(), target);
        }

        Ok(Some(info))
    }

    /// gets flist information from name. the flist must be of type flist or symlink
    pub fn get<T>(&self, flist: T) -> Result<Flist>
    where
        T: AsRef<str>,
    {
        let url = format!(
            "{}/api/flist/{}/{}/light",
            self.get_working_hub(),
            self.name,
            flist.as_ref()
        );

        let response = get(&url)?;
        let mut info: Flist = match response.status() {
            StatusCode::OK => response.json()?,
            s => bail!("failed to get flist info: {}", s),
        };
        info.url = format!(
            "{}/{}/{}",
            self.get_working_hub(),
            self.name,
            flist.as_ref()
        );
        Ok(info)
    }
}

impl Flist {
    /// tag_link return the target repo and tag name for
    /// a taglink flist. Panic if the flist is not of type TagLink
    pub fn tag_link(self) -> (String, String) {
        if self.kind != Kind::TagLink {
            panic!("invalid flist type must be a taglink");
        }

        let target = self.target.unwrap();
        let parts: Vec<&str> = target.split('/').collect();
        assert_eq!(parts.len(), 3, "link must be 3 parts");
        assert_eq!(parts[1], "tags");

        (parts[0].to_owned(), parts[2].to_owned())
    }

    /// download the flist to `out`, panics if flist kind is
    /// not regular or symlink
    pub fn download<T>(&self, out: T) -> Result<()>
    where
        T: AsRef<Path>,
    {
        match self.kind {
            Kind::Regular | Kind::Symlink => {}
            _ => {
                panic!("invalid flist type, must be regular or symlink")
            }
        }

        let mut file = OpenOptions::new().write(true).create(true).open(out)?;

        let mut response = get(&self.url)?;
        if !response.status().is_success() {
            bail!("failed to download flist: {}", response.status());
        }

        copy(&mut response, &mut file)?;
        Ok(())
    }

    /// write the flist info (in json format) to out
    pub fn write<T>(&self, out: T) -> Result<()>
    where
        T: AsRef<Path>,
    {
        write(out, serde_json::to_vec(self)?)?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn test_get_flist() -> Result<()> {
        let repo = Repo::new(String::from("azmy"))?;
        let flist = repo.get("test.flist")?;
        assert_eq!(flist.name, "test.flist");
        assert_eq!(flist.kind, Kind::Regular);
        assert!(flist.url.ends_with("/azmy/test.flist"));

        Ok(())
    }

    #[test]
    fn test_list_tag() -> Result<()> {
        let repo = Repo::new(String::from("tf-autobuilder"))?;

        let list = repo.list_tag("3b51aa5")?;

        assert!(list.is_some());
        let list = list.unwrap();

        assert_ne!(list.len(), 0);

        assert!(repo.list_tag("wrong")?.is_none());

        Ok(())
    }

    #[test]
    fn test_download_flist() -> Result<()> {
        let repo = Repo::new(String::from("azmy"))?;
        let flist = repo.get("test.flist")?;
        let temp = "/tmp/hub-download-test.flist";
        flist.download(temp)?;

        // comput hash sum of the downloaded file
        // and compare it with flist hash info
        use std::process::Command;
        let output = Command::new("md5sum").arg(temp).output()?;
        let output = String::from_utf8(output.stdout)?;
        let line: Vec<&str> = output.split_whitespace().collect();

        assert_eq!(line[0], &flist.md5);
        let _ = std::fs::remove_file(temp);
        Ok(())
    }

    #[test]
    fn test_list_repo() -> Result<()> {
        let repo = Repo::new(String::from("azmy"))?;
        let lists = repo.list()?;

        let mut found: Option<&Flist> = None;
        for flist in lists.iter() {
            if flist.name == "test.flist" {
                found = Some(flist);
                break;
            }
        }

        let found = found.unwrap();
        assert_eq!(found.name, "test.flist");
        assert!(found.url.ends_with("/azmy/test.flist"));

        Ok(())
    }
}
