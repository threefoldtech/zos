use crate::executor;

use std::path::{Path, PathBuf};

use serde::{Deserialize, Serialize};

const USB_SUBSYSTEM: &'static str = "block:scsi:usb:pci";
const FLOPPY_SUBSYSTEM: &'static str = "block:platform";
const CD_TYPE: &'static str = "rom";
const PARTITION_TYPE: &'static str = "part";

const CACHE_LABEL: &'static str = "sp_zos_cache";

const CACHE_DIR: &'static str = "/var/cache";
const SHARED_CACHE: &'static str = "/var/cache/zerofs";
const LOG_PATH: &'static str = "/var/log";

const CONTAINER_DIR: &'static str = "/var/cache/containers";
const VM_DIR: &'static str = "/var/cache/vms";

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
            // ignore USB and CD_ROM and floppy
            .filter(|bd| {
                bd.node_info.subsystems != USB_SUBSYSTEM
                    && bd.node_info._type != CD_TYPE
                    && bd.node_info.subsystems != FLOPPY_SUBSYSTEM
            })
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

        let mut dm = DiskManager {
            executor,

            device_nodes: nodes,
        };

        dm.ensure_cache().expect("Failed to mount cache");

        dm
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
            let mut base = PathBuf::from(DISK_LABEL_PATH);
            base.push(CACHE_LABEL);

            base
        };

        debug!("Repairing disk {:?}", &disk);
        if !self.executor.btrfs_repair(&disk)? {
            info!("{} not mounted, look for empty disk", CACHE_LABEL);
            // find empty disk, panic if not found
            let empty_disk = self.find_empty_disk()?.expect("Failed to find empty disk");
            // prepare disk
            debug!("partitioning disk {}", empty_disk.name);
            self.executor.partition_disk(&empty_disk)?;
            debug!("Creating btrfs partition on {}", empty_disk.name);
            self.executor.create_fs(&empty_disk)?;
            let disk_part = match self.get_child_partition(&empty_disk)? {
                None => {
                    error!("Could not get child partition of just partitioned disk");
                    return Err(executor::Error::IOError(std::io::Error::from(
                        std::io::ErrorKind::NotFound,
                    )));
                }
                Some(disk) => {
                    debug!("disk partition {}", disk.name);
                    disk
                }
            };
            debug!(
                "Creating btrfs file system on {} with label {}",
                disk_part.name, CACHE_LABEL
            );
            self.executor.make_fs(&disk_part, CACHE_LABEL)?;
        }

        debug!(
            "mounting {} to {}",
            disk.to_str().unwrap(),
            &mount_dir.display()
        );
        self.executor.mount(&disk, &mount_dir, Some("btrfs"))?;

        if self.executor.is_directory_mountpoint(mount_dir)? {
            info!("Create and mount subvolume for {}", CACHE_LABEL);

            // cache subvol
            debug!("Mounting cache dir");
            let cache_path = {
                let mut buf = PathBuf::from(mount_dir);
                buf.push("cache");

                buf
            };
            self.executor.create_btrfs_subvol(&cache_path)?;
            self.executor
                .mount(&cache_path, &Path::new(CACHE_DIR), Some("btrfs"))?;

            // cleanup old dirs
            debug!("Cleanup old vm and container working dirs");
            for p in &[CONTAINER_DIR, VM_DIR] {
                let dir = Path::new(p);
                if !dir.exists() {
                    continue;
                }
                trace!("listing contents of {}", dir.display());
                let vols = self.executor.list_dir(dir)?;
                for vol in vols {
                    trace!("Deleting btrfs subvol in {}", &vol.path().display());
                    self.executor.delete_btrfs_subvol(&vol.path())?;
                    trace!("Deleting directory {}", &vol.path().display());
                    self.executor.delete_dir(&vol.path())?;
                }
            }

            // log subvol
            debug!("Mounting log dir");
            let mut log_path = {
                let mut buf = PathBuf::from(mount_dir);
                buf.push("logs");

                buf
            };
            self.executor.create_btrfs_subvol(&log_path)?;
            let time = chrono::Local::now();
            let log_dir_name = format!("log-{}", time.format("%Y%m%d-%H%M"));
            log_path.push(log_dir_name);
            self.executor.create_btrfs_subvol(&log_path)?;

            debug!(
                "Copy old logs ({}) to log dir ({})",
                LOG_PATH,
                log_path.as_path().display()
            );
            let log_entries = self.executor.list_dir(Path::new(LOG_PATH))?;
            for log_entry in log_entries {
                trace!(
                    "Copy {} to {}",
                    &log_entry.path().display(),
                    log_path.as_path().display()
                );
                self.executor.copy_dir(&log_entry.path(), &log_path)?;
            }
            debug!(
                "Mounting logs ({}) to {}",
                LOG_PATH,
                log_path.as_path().display()
            );
            self.executor
                .mount(&log_path, Path::new(LOG_PATH), Some("btrfs"))?;

            // vm inside vm stuff, ignore for now
            //info!("Try to mount shared cache");
            //let shared_cache = Path::new(SHARED_CACHE);
            //self.executor.make_dir(shared_cache)?;
            //match self.executor.mount("zoscache", shared_cache, Some("9p"))? {
            //    true => debug!("Shared cache mounted"),
            //    false => debug!("Failed to mount shared cache"),
            //}

            info!("Finished setting up cache");
        }

        Ok(())
    }

    fn list_nodes(&self) -> executor::Result<Vec<Node>> {
        let disks: RootInterface = serde_json::from_str(&self.executor.lsblk()?)
            .expect("Failed to interpret lsblk output");

        let filtered_disks: Vec<Blockdevice> = disks
            .blockdevices
            .into_iter()
            // ignore USB and CD_ROM and floppy
            .filter(|bd| {
                bd.node_info.subsystems != USB_SUBSYSTEM
                    && bd.node_info._type != CD_TYPE
                    && bd.node_info.subsystems != FLOPPY_SUBSYSTEM
            })
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
        debug!("Looking for empty disks");
        let disks: RootInterface = serde_json::from_str(&self.executor.lsblk()?)
            .expect("Failed to interpret lsblk output");

        trace!("Parsed lsblk output");

        let filtered_disks: Vec<Blockdevice> = disks
            .blockdevices
            .into_iter()
            // ignore USB and CD_ROM
            .filter(|bd| {
                bd.node_info.subsystems != USB_SUBSYSTEM
                    && bd.node_info._type != CD_TYPE
                    && bd.node_info.subsystems != FLOPPY_SUBSYSTEM
            })
            .collect();

        for disk in filtered_disks {
            trace!("disk name: {}", disk.node_info.name);
            let mut disk_empty = true;
            match disk.children {
                None => {
                    trace!("Disk has no children");
                    return Ok(Some(disk.node_info));
                }
                Some(children) => {
                    for child in children {
                        trace!("Disk child {}", child.node_info.name);
                        if child.node_info._type == PARTITION_TYPE {
                            trace!("Disk not empty");
                            disk_empty = false;
                            break;
                        }
                    }
                    if disk_empty {
                        trace!("Found empty disk {}", disk.node_info.name);
                        return Ok(Some(disk.node_info));
                    }
                }
            }
        }
        Ok(None)
    }

    fn get_child_partition(&self, node: &Node) -> executor::Result<Option<Node>> {
        debug!("Looking for child partition of {}", node.name);
        let disks: RootInterface = serde_json::from_str(&self.executor.lsblk()?)
            .expect("Failed to interpret lsblk output");

        trace!("Parsed lsblk output");

        if let Some(disk) = disks
            .blockdevices
            .into_iter()
            .filter(|bd| bd.node_info.name == node.name)
            .nth(0)
        {
            if let Some(children) = disk.children {
                return Ok(children.into_iter().map(|part| part.node_info).nth(0));
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
    pub name: String,
    kname: String,
    #[serde(default)]
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
    //TODO: lsblk 2.2 -> string, lsblk 2.33.2 -> u64
    //ra: u64,
    //ro: bool,
    //rm: bool,
    //hotplug: bool,
    //model: Option<String>,
    //serial: Option<String>,
    //size: u64,
    //state: Option<String>,
    //owner: Option<String>,
    //group: Option<String>,
    //mode: String,
    //alignment: u64,
    //#[serde(rename = "min-io")]
    //min_io: u64,
    //#[serde(rename = "opt-io")]
    //opt_io: u64,
    //#[serde(rename = "phy-sec")]
    //phy_sec: u64,
    //#[serde(rename = "log-sec")]
    //log_sec: u64,
    //rota: bool,
    //sched: String,
    //#[serde(rename = "rq-size")]
    //rq_size: u64,
    #[serde(rename = "type")]
    _type: String,
    //#[serde(rename = "disc-aln")]
    //disc_aln: u64,
    //#[serde(rename = "disc-gran")]
    //disc_gran: u64,
    //#[serde(rename = "disc-max")]
    //disc_max: u64,
    //#[serde(rename = "disc-zero")]
    //disc_zero: bool,
    //wsame: u64,
    //wwn: Option<String>,
    //rand: bool,
    //pkname: Option<String>,
    //hctl: Option<String>,
    //tran: Option<String>,
    subsystems: String,
    //rev: Option<String>,
    //vendor: Option<String>,
    //zoned: String,
}

#[derive(Debug, Serialize, Deserialize)]
struct RootInterface {
    blockdevices: Vec<Blockdevice>,
}

#[cfg(test)]
mod tests {
    use super::RootInterface;

    #[test]
    fn parse_lsblk_output() {
        let _: RootInterface =
            serde_json::from_str(TEST_OUTPUT).expect("Failed to interpret lsblk output");
    }
    const TEST_OUTPUT: &'static str = r#"{
   "blockdevices": [
      {"name": "sda", "kname": "sda", "maj:min": "8:0", "fstype": null, "mountpoint": null, "label": null, "uuid": null, "parttype": null, "partlabel": null, "partuuid": null, "partflags": null, "ra": "128", "ro": "0", "rm": "0", "hotplug": "0", "model": "QEMU HARDDISK   ", "serial": null, "size": "528482304", "state": "running", "owner": "root", "group": "disk", "mode": "brw-rw----", "alignment": "0", "min-io": "512", "opt-io": "0", "phy-sec": "512", "log-sec": "512", "rota": "1", "sched": "cfq", "rq-size": "128", "type": "disk", "disc-aln": "0", "disc-gran": "512", "disc-max": "2147450880", "disc-zero": "0", "wsame": "0", "wwn": null, "rand": "1", "pkname": null, "hctl": "0:0:0:0", "tran": "ata", "subsystems": "block:scsi:pci", "rev": "2.5+", "vendor": "ATA     ",
         "children": [
            {"name": "sda1", "kname": "sda1", "maj:min": "8:1", "fstype": null, "mountpoint": null, "label": null, "uuid": null, "parttype": null, "partlabel": null, "partuuid": null, "partflags": null, "ra": "128", "ro": "0", "rm": "0", "hotplug": "0", "model": null, "serial": null, "size": "528450048", "state": null, "owner": "root", "group": "disk", "mode": "brw-rw----", "alignment": "0", "min-io": "512", "opt-io": "0", "phy-sec": "512", "log-sec": "512", "rota": "1", "sched": "cfq", "rq-size": "128", "type": "part", "disc-aln": "0", "disc-gran": "512", "disc-max": "2147450880", "disc-zero": "0", "wsame": "0", "wwn": null, "rand": "1", "pkname": "sda", "hctl": null, "tran": null, "subsystems": "block:scsi:pci", "rev": null, "vendor": null}
         ]
      },
      {"name": "vda", "kname": "vda", "maj:min": "254:0", "fstype": null, "mountpoint": null, "label": null, "uuid": null, "parttype": null, "partlabel": null, "partuuid": null, "partflags": null, "ra": "128", "ro": "0", "rm": "0", "hotplug": "0", "model": null, "serial": null, "size": "53687091200", "state": null, "owner": "root", "group": "disk", "mode": "brw-rw----", "alignment": "0", "min-io": "512", "opt-io": "0", "phy-sec": "512", "log-sec": "512", "rota": "1", "sched": "mq-deadline", "rq-size": "256", "type": "disk", "disc-aln": "0", "disc-gran": "0", "disc-max": "0", "disc-zero": "0", "wsame": "0", "wwn": null, "rand": "0", "pkname": null, "hctl": null, "tran": null, "subsystems": "block:virtio:pci", "rev": null, "vendor": "0x1af4"},
      {"name": "vdb", "kname": "vdb", "maj:min": "254:16", "fstype": null, "mountpoint": null, "label": null, "uuid": null, "parttype": null, "partlabel": null, "partuuid": null, "partflags": null, "ra": "128", "ro": "0", "rm": "0", "hotplug": "0", "model": null, "serial": null, "size": "53687091200", "state": null, "owner": "root", "group": "disk", "mode": "brw-rw----", "alignment": "0", "min-io": "512", "opt-io": "0", "phy-sec": "512", "log-sec": "512", "rota": "1", "sched": "mq-deadline", "rq-size": "256", "type": "disk", "disc-aln": "0", "disc-gran": "0", "disc-max": "0", "disc-zero": "0", "wsame": "0", "wwn": null, "rand": "0", "pkname": null, "hctl": null, "tran": null, "subsystems": "block:virtio:pci", "rev": null, "vendor": "0x1af4"},
      {"name": "vdc", "kname": "vdc", "maj:min": "254:32", "fstype": null, "mountpoint": null, "label": null, "uuid": null, "parttype": null, "partlabel": null, "partuuid": null, "partflags": null, "ra": "128", "ro": "0", "rm": "0", "hotplug": "0", "model": null, "serial": null, "size": "53687091200", "state": null, "owner": "root", "group": "disk", "mode": "brw-rw----", "alignment": "0", "min-io": "512", "opt-io": "0", "phy-sec": "512", "log-sec": "512", "rota": "1", "sched": "mq-deadline", "rq-size": "256", "type": "disk", "disc-aln": "0", "disc-gran": "0", "disc-max": "0", "disc-zero": "0", "wsame": "0", "wwn": null, "rand": "0", "pkname": null, "hctl": null, "tran": null, "subsystems": "block:virtio:pci", "rev": null, "vendor": "0x1af4"},
      {"name": "vdd", "kname": "vdd", "maj:min": "254:48", "fstype": null, "mountpoint": null, "label": null, "uuid": null, "parttype": null, "partlabel": null, "partuuid": null, "partflags": null, "ra": "128", "ro": "0", "rm": "0", "hotplug": "0", "model": null, "serial": null, "size": "53687091200", "state": null, "owner": "root", "group": "disk", "mode": "brw-rw----", "alignment": "0", "min-io": "512", "opt-io": "0", "phy-sec": "512", "log-sec": "512", "rota": "1", "sched": "mq-deadline", "rq-size": "256", "type": "disk", "disc-aln": "0", "disc-gran": "0", "disc-max": "0", "disc-zero": "0", "wsame": "0", "wwn": null, "rand": "0", "pkname": null, "hctl": null, "tran": null, "subsystems": "block:virtio:pci", "rev": null, "vendor": "0x1af4"},
      {"name": "vde", "kname": "vde", "maj:min": "254:64", "fstype": null, "mountpoint": null, "label": null, "uuid": null, "parttype": null, "partlabel": null, "partuuid": null, "partflags": null, "ra": "128", "ro": "0", "rm": "0", "hotplug": "0", "model": null, "serial": null, "size": "53687091200", "state": null, "owner": "root", "group": "disk", "mode": "brw-rw----", "alignment": "0", "min-io": "512", "opt-io": "0", "phy-sec": "512", "log-sec": "512", "rota": "1", "sched": "mq-deadline", "rq-size": "256", "type": "disk", "disc-aln": "0", "disc-gran": "0", "disc-max": "0", "disc-zero": "0", "wsame": "0", "wwn": null, "rand": "0", "pkname": null, "hctl": null, "tran": null, "subsystems": "block:virtio:pci", "rev": null, "vendor": "0x1af4"}
   ]
}"#;
}
