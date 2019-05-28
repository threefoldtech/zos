use crate::executor;

use std::path::Path;

use serde::{Deserialize, Serialize};

const USB_SUBSYSTEM: &'static str = "block:scsi:usb:pci";
const CD_TYPE: &'static str = "rom";
const PARTITION_TYPE: &'static str = "part";
const CACHE_LABEL: &'static str = "sp_zos_cache";

const MOUNT_DIR_STR: &'static str = "/mnt/storagepools/sp_zos_cache";
const DISK_LABEL_PATH: &'static str = "/dev/disk/by-label";

pub struct DiskManager<'a> {
    executor: &'a mut dyn executor::Executor,

    device_nodes: Vec<Node>,
}

impl<'a> DiskManager<'a> {
    pub fn new(executor: &'a mut dyn executor::Executor) -> Self {
        let disks: RootInterface =
            serde_json::from_str(&executor.lsblk().expect("Failed to get disk list"))
                .expect("Failed to interpret lsblk output");

        let filtered_disks: Vec<Blockdevice> = disks
            .blockdevices
            .into_iter()
            // ignore USB and CD_ROM
            .filter(|bd| bd.node_info.subsystems != USB_SUBSYSTEM && bd.node_info._type != CD_TYPE)
            .collect();

        let mut nodes = Vec::new();
        for disk in filtered_disks {
            // push disk itself
            nodes.push(disk.node_info);
            // push disk children
            nodes.extend(match disk.children {
                Some(children) => children.into_iter().map(|child| child.node_info).collect(),
                _ => vec![],
            });
        }

        DiskManager {
            executor,

            device_nodes: nodes,
        }
    }

    /// list all free nodes on the system. A free node is defined as not having a filesystem, and,
    /// in case of a disk, not being partitioned.
    pub fn list_free_nodes(&self) -> Vec<&Node> {
        let mut free_nodes = Vec::new();

        for node in self.device_nodes.iter().filter(|node| {
            node.fstype.is_none()
                && ((node._type != PARTITION_TYPE && node.ptuuid.is_none())
                    || node._type == PARTITION_TYPE)
        }) {
            free_nodes.push(node);
        }

        free_nodes
    }

    fn ensure_cache(&mut self) -> executor::Result<()> {
        // TODO: check if this is the proper place for this
        info!("Ensuring cache exists and is mounted");

        let mount_dir = Path::new(MOUNT_DIR_STR);

        self.executor.make_dir(mount_dir)?;

        debug!(
            "Checking if directory ({}) is already mounted",
            mount_dir.display()
        );
        if self.executor.is_directory_mountpoint(mount_dir)? {
            error!("{} is already mounted!", mount_dir.display());
            // TODO: proper error
            return Err(executor::Error::IOError(std::io::Error::from(
                std::io::ErrorKind::InvalidData,
            )));
        }

        let disk = {
            let mut base = std::path::PathBuf::from(DISK_LABEL_PATH);
            base.push(CACHE_LABEL);

            base
        };

        debug!("Repairing disk {:?}", &disk);
        if !self
            .executor
            .btrfs_repair(&disk)
            .expect("Failed btrfs repair command")
        {
            info!("{} not mounted, look for empty disk", CACHE_LABEL);
            // find empty disk, panic if not found
            let empty_disk = self.find_empty_disk()?.expect("Failed to find empty disk");
            // prepare disk
            self.executor.partition_disk(&empty_disk)?;
            self.executor.create_fs(&empty_disk)?;
            self.executor.make_fs(&empty_disk, CACHE_LABEL)?;
        }

        self.executor
            .mount(disk.to_str().unwrap(), &mount_dir, None)?;

        Ok(())
    }

    fn list_nodes(&self) -> executor::Result<Vec<Node>> {
        let disks: RootInterface = serde_json::from_str(&self.executor.lsblk()?)
            .expect("Failed to interpret lsblk output");

        let filtered_disks: Vec<Blockdevice> = disks
            .blockdevices
            .into_iter()
            // ignore USB and CD_ROM
            .filter(|bd| bd.node_info.subsystems != USB_SUBSYSTEM && bd.node_info._type != CD_TYPE)
            .collect();

        let mut nodes = Vec::new();
        for disk in filtered_disks {
            // push disk itself
            nodes.push(disk.node_info);
            // push disk children
            nodes.extend(match disk.children {
                Some(children) => children.into_iter().map(|child| child.node_info).collect(),
                _ => vec![],
            });
        }

        Ok(nodes)
    }

    fn find_empty_disk(&self) -> executor::Result<Option<Node>> {
        let disks: RootInterface = serde_json::from_str(&self.executor.lsblk()?)
            .expect("Failed to interpret lsblk output");

        let filtered_disks: Vec<Blockdevice> = disks
            .blockdevices
            .into_iter()
            // ignore USB and CD_ROM
            .filter(|bd| bd.node_info.subsystems != USB_SUBSYSTEM && bd.node_info._type != CD_TYPE)
            .collect();

        for disk in filtered_disks {
            let mut disk_empty = true;
            match disk.children {
                None => continue,
                Some(children) => {
                    for child in children {
                        if child.node_info._type == PARTITION_TYPE {
                            disk_empty = false;
                            break;
                        }
                        if disk_empty {
                            return Ok(Some(disk.node_info));
                        }
                    }
                }
            }
        }
        Ok(None)
    }
}

impl<'a> std::fmt::Debug for DiskManager<'a> {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        write!(f, "{:?}", self.device_nodes)
    }
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Blockdevice {
    #[serde(flatten)]
    node_info: Node,
    children: Option<Vec<Partition>>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Partition {
    #[serde(flatten)]
    node_info: Node,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Node {
    name: String,
    kname: String,
    pub path: String,
    #[serde(rename = "maj:min")]
    maj_min: String,
    fsavail: Option<String>,
    fssize: Option<String>,
    fstype: Option<String>,
    fsused: Option<String>,
    #[serde(rename = "fsuse%")]
    _fsuse: Option<String>,
    mountpoint: Option<String>,
    label: Option<String>,
    uuid: Option<String>,
    ptuuid: Option<String>,
    pttype: Option<String>,
    parttype: Option<String>,
    partlabel: Option<String>,
    partuuid: Option<String>,
    partflags: Option<String>,
    ra: u64,
    ro: bool,
    rm: bool,
    hotplug: bool,
    model: Option<String>,
    serial: Option<String>,
    size: u64,
    state: Option<String>,
    owner: Option<String>,
    group: Option<String>,
    mode: String,
    alignment: u64,
    #[serde(rename = "min-io")]
    min_io: u64,
    #[serde(rename = "opt-io")]
    opt_io: u64,
    #[serde(rename = "phy-sec")]
    phy_sec: u64,
    #[serde(rename = "log-sec")]
    log_sec: u64,
    rota: bool,
    sched: String,
    #[serde(rename = "rq-size")]
    rq_size: u64,
    #[serde(rename = "type")]
    _type: String,
    #[serde(rename = "disc-aln")]
    disc_aln: u64,
    #[serde(rename = "disc-gran")]
    disc_gran: u64,
    #[serde(rename = "disc-max")]
    disc_max: u64,
    #[serde(rename = "disc-zero")]
    disc_zero: bool,
    wsame: u64,
    wwn: Option<String>,
    rand: bool,
    pkname: Option<String>,
    hctl: Option<String>,
    tran: Option<String>,
    subsystems: String,
    rev: Option<String>,
    vendor: Option<String>,
    zoned: String,
}

#[derive(Debug, Serialize, Deserialize)]
struct RootInterface {
    blockdevices: Vec<Blockdevice>,
}
