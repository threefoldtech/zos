use failure::Error;
use reqwest::get; 
use serde::Deserialize;

const HUB: &str = "https://hub.grid.tf";

type Result<T> = std::result::Result<T, Error>;

struct Repo {
    base: String,
    name: String,
}

#[derive(Deserialize)]
struct Flist {
    #[serde(rename = "type")]
    pub kind: String,
    pub updated: u64,
    pub size: u64,
    pub md5: String,
    pub name: String,
    #[serde(default)]
    pub target: String,

    #[serde(skip)]
    url: String
}

impl Repo {
    pub fn new(name: String) -> Repo {
        Repo{
            base: String::from(HUB), 
            name: name
        }
    }

    pub fn get<T>(&self, flist: T) -> Result<Flist>
        where T: AsRef<str> {

        let url = format!("{}/api/flist/{}/{}/light", self.base, self.name, flist.as_ref());
        let mut info: Flist = get(&url)?
            .json()?;

        info.url = format!("{}/{}/{}", self.base, self.name, flist.as_ref());
        Ok(info)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn test_get_flist() -> Result<()> {
        let repo = Repo::new(String::from("azmy"));
        let flist = repo.get("test.flist")?;
        
        assert_eq!(flist.name, "test.flist");
        assert_eq!(flist.kind, "regular");
        assert_eq!(flist.url, "https://hub.grid.tf/azmy/test.flist");

        Ok(())
    }
}