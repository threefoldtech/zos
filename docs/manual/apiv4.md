# ZOS APIv4

The idea is that instead of working with a single deployment endpoint that deals with a full deployment that we instead do smaller and separate transactions to CRUD operations for components

## Main RPC endpoints

Users can create any project with a specific name that includes any existent components

- `v4/zos.project.get`  (project name) -> gets a specific project using its name

```json
{
  "command": "v4/zos.project.get",
  "data": {
    "name": "project name"
  },
  "result": {
    "status": "ok", 
    "error": "",
    "data": {
      "name": "project name",
      "twin_id": "user twin id",
      "contracts_id": "project contracts id",
      "metadata": "project metadata",
      "description": "project description",
      "expiration": "project expiration",
      "signature": "user signature"
    }
  }
}
```

- `v4/zos.project.list` -> lists all projects for the user that made the calls

```json
{
  "command": "v4/zos.project.list",
  "data": {},
  "result": {
    "status": "ok", 
    "error": "",
    "data": [{
      "name": "project name",
      "twin_id": "user twin id",
      "contracts_id": "project contracts id",
      "metadata": "project metadata",
      "description": "project description",
      "expiration": "project expiration",
      "signature": "user signature"
    }, ...]
  }
}
```

## Endpoints for components

- `v4/zos.project.<component>.create`  (project name, component date)

- `v4/zos.project.<component>.restart`  (project name, component date)

- `v4/zos.project.<component>.get`   (project name, component name)

- `v4/zos.project.<component>.update`  (project name, component data)

- `v4/zos.project.<component>.delete`   (project name, component name)

## Components

### Network

- `v4/zos.project.network.create`  (project name, network date)

```json
{
  "command": "v4/zos.project.network.create",
  "data": {
    "project_name": "project name",
    "name": "network name",
    "version": "deployment version",
    "description": "network description",
    "metadata": "network metadata (UserAccessIP, PrivateKey, PublicNodeID)",
    "ip_range": "network ip range",
    "subnet": "network subnet",
    "wireguard_private_key": "network private key",
    "wireguard_listen_port": "network listen port",
    "peers": ["network list of peers"]
  },
  "result": {
    "status": "created", 
    "error": ""
  }
}
```

- `v4/zos.project.network.update`  (project name, network date)

```json
{
  "command": "v4/zos.project.network.update",
  "data": {
    "project_name": "project name",
    "name": "network name",
    "version": "deployment version",
    "description": "network description",
    "metadata": "network metadata (UserAccessIP, PrivateKey, PublicNodeID)",
    "ip_range": "network ip range",
    "subnet": "network subnet",
    "wireguard_private_key": "network private key",
    "wireguard_listen_port": "network listen port",
    "peers": ["network list of peers"]
  },
  "result": {
    "status": "ok", 
    "error": ""
  }
}
```

- `v4/zos.project.network.get`   (project name, network name)

```json
{
  "command": "v4/zos.project.network.get",
  "data": {
    "project_name": "project name",
    "name": "network name"
  },
  "result": {
    "status": "ok", 
    "error": "",
    "data": {
      "project_name": "project name",
      "name": "network name",
      "version": "deployment version",
      "description": "network description",
      "metadata": "network metadata (UserAccessIP, PrivateKey, PublicNodeID)",
      "ip_range": "network ip range",
      "subnet": "network subnet",
      "wireguard_private_key": "network private key",
      "wireguard_listen_port": "network listen port",
      "peers": ["network list of peers"]
    }
  }
}
```

- `v4/zos.project.network.delete`   (project name, network name)

```json
{
  "command": "v4/zos.project.network.delete",
  "data": {
    "project_name": "project name",
    "name": "network name"
  },
  "result": {
    "status": "deleted", 
    "error": ""
  }
}
```

### VM

- `v4/zos.project.vm.create`  (project name, vm date)

```json
{
  "command": "v4/zos.project.vm.create",
  "data": {
    "project_name": "project name",
    "name": "vm name",
    "version": "deployment version",
    "description": "vm description",
    "flist": "vm flist",
    "network": "vm network",
    "size": "disk size in GB",
    "capacity": "cpu and memory",
    "mounts": [{
      "name": "mount name",
      "point": "mount point"
    }, ...],
    "entrypoint": "vm entry point",
    "env": {
      "key": "value"
    },
    "corex": "vm corex",
    "gpu": ["vm list of gpus"]
  },
  "result": {
    "status": "created", 
    "error": ""
  }
}
```

- `v4/zos.project.vm.update`  (project name, vm date)

```json
{
  "command": "v4/zos.project.vm.update",
  "data": {
    "project_name": "project name",
    "name": "vm name",
    "version": "deployment version",
    "description": "vm description",
    "flist": "vm flist",
    "network": "vm network",
    "size": "disk size in GB",
    "capacity": "cpu and memory",
    "mounts": [{
      "name": "mount name",
      "point": "mount point"
    }, ...],
    "entrypoint": "vm entry point",
    "env": {
      "key": "value"
    },
    "corex": "vm corex",
    "gpu": ["vm list of gpus"]
  },
  "result": {
    "status": "ok", 
    "error": ""
  }
}
```

- `v4/zos.project.vm.get`  (project name, vm date)

```json
{
  "command": "v4/zos.project.vm.get",
  "data": {
    "project_name": "project name",
    "name": "vm name"
  },
  "result": {
    "status": "ok", 
    "error": "",
    "data": {
      "project_name": "project name",
      "name": "vm name",
      "version": "deployment version",
      "version": "deployment version",
      "description": "vm description",
      "flist": "vm flist",
      "network": "vm network",
      "size": "disk size in GB",
      "capacity": "cpu and memory",
      "mounts": [{
        "name": "mount name",
        "point": "mount point"
      }, ...],
      "entrypoint": "vm entry point",
      "env": {
        "key": "value"
      },
      "corex": "vm corex",
      "gpu": ["vm list of gpus"]
    }
  }
}
```

- `v4/zos.project.vm.delete`  (project name, vm date)

```json
{
  "command": "v4/zos.project.vm.delete",
  "data": {
    "project_name": "project name",
    "name": "vm name"
  },
  "result": {
    "status": "deleted", 
    "error": "",
    "data": {}
  }
}
```

### Disk

- `v4/zos.project.disk.create`  (project name, disk date)

```json
{
  "command": "v4/zos.project.disk.create",
  "data": {
    "project_name": "project name",
    "name": "disk name",
    "version": "deployment version",
    "description": "disk description",
    "size": "disk size"
  },
  "result": {
    "status": "created", 
    "error": ""
  }
}
```

- `v4/zos.project.disk.update`  (project name, disk date)

```json
{
  "command": "v4/zos.project.disk.update",
  "data": {
    "project_name": "project name",
    "name": "disk name",
    "version": "deployment version",
    "description": "disk description",
    "size": "disk size"
  },
  "result": {
    "status": "ok", 
    "error": ""
  }
}
```

- `v4/zos.project.disk.get`  (project name, disk date)

```json
{
  "command": "v4/zos.project.disk.get",
  "data": {
    "project_name": "project name",
    "name": "disk name",
  },
  "result": {
    "status": "ok", 
    "error": "",
    "data": {
      "project_name": "project name",
      "name": "disk name",
      "version": "deployment version",
      "description": "disk description",
      "size": "disk size"
    }
  }
}
```

- `v4/zos.project.disk.delete`  (project name, disk date)

```json
{
  "command": "v4/zos.project.disk.delete",
  "data": {
    "project_name": "project name",
    "name": "disk name",
  },
  "result": {
    "status": "deleted", 
    "error": "",
    "data": {}
  }
}
```

### ZDB

- `v4/zos.project.zdb.create`  (project name, zdb date)

```json
{
  "command": "v4/zos.project.zdb.create",
  "data": {
    "project_name": "project name",
    "name": "zdb name",
    "version": "deployment version",
    "mode": "zdb mode",
    "size": "zdb size in GB",
    "password": "",
    "public": "if zdb gets a public ip6"
  },
  "result": {
    "status": "created", 
    "error": ""
  }
}
```

- `v4/zos.project.zdb.update`  (project name, zdb date)

```json
{
  "command": "v4/zos.project.zdb.update",
  "data": {
    "project_name": "project name",
    "name": "zdb name",
    "version": "deployment version",
    "mode": "zdb mode",
    "size": "zdb size in GB",
    "password": "",
    "public": "if zdb gets a public ip6"
  },
  "result": {
    "status": "ok", 
    "error": ""
  }
}
```

- `v4/zos.project.zdb.get`  (project name, zdb date)

```json
{
  "command": "v4/zos.project.zdb.get",
  "data": {
    "project_name": "project name",
    "name": "zdb name"
  },
  "result": {
    "status": "ok", 
    "error": "",
    "data": {
      "project_name": "project name",
      "name": "zdb name",
      "version": "deployment version",
      "mode": "zdb mode",
      "size": "zdb size in GB",
      "password": "",
      "public": "if zdb gets a public ip6"
    }
  }
}
```

- `v4/zos.project.zdb.delete`  (project name, zdb date)

```json
{
  "command": "v4/zos.project.zdb.delete",
  "data": {
    "project_name": "project name",
    "name": "zdb name"
  },
  "result": {
    "status": "ok", 
    "error": "",
    "data": {}
  }
}
```

### QSFS

- `v4/zos.project.qsfs.create`  (project name, qsfs date)

```json
{
  "command": "v4/zos.project.qsfs.create",
  "data": {
    "project_name": "project name",
    "name": "qsfs name",
    "version": "deployment version",
    "cache": "qsfs cache",
    "minimal_shards": "qsfs minimal shards",
    "expected_shards": "qsfs expected shards",
    "redundant_groups": "qsfs redundant groups",
    "redundant_nodes": "qsfs redundant nodes",
    "max_zdb_data_dir_size": "qsfs max zdb data dir size",
    "encryption": "qsfs encryption",
    "metadata": "qsfs metadata",
    "groups": "qsfs groups",
    "compression": "qsfs compression"
  },
  "result": {
    "status": "created", 
    "error": ""
  }
}
```

- `v4/zos.project.qsfs.update`  (project name, qsfs date)

```json
{
  "command": "v4/zos.project.qsfs.update",
  "data": {
    "project_name": "project name",
    "name": "qsfs name",
    "version": "deployment version",
    "cache": "qsfs cache",
    "minimal_shards": "qsfs minimal shards",
    "expected_shards": "qsfs expected shards",
    "redundant_groups": "qsfs redundant groups",
    "redundant_nodes": "qsfs redundant nodes",
    "max_zdb_data_dir_size": "qsfs max zdb data dir size",
    "encryption": "qsfs encryption",
    "metadata": "qsfs metadata",
    "groups": "qsfs groups",
    "compression": "qsfs compression"
  },
  "result": {
    "status": "ok", 
    "error": ""
  }
}
```

- `v4/zos.project.qsfs.get`  (project name, qsfs date)

```json
{
  "command": "v4/zos.project.qsfs.get",
  "data": {
    "project_name": "project name",
    "name": "qsfs name",
  },
  "result": {
    "status": "ok", 
    "error": "",
    "data": {
      "project_name": "project name",
      "name": "qsfs name",
      "version": "deployment version",
      "cache": "qsfs cache",
      "minimal_shards": "qsfs minimal shards",
      "expected_shards": "qsfs expected shards",
      "redundant_groups": "qsfs redundant groups",
      "redundant_nodes": "qsfs redundant nodes",
      "max_zdb_data_dir_size": "qsfs max zdb data dir size",
      "encryption": "qsfs encryption",
      "metadata": "qsfs metadata",
      "groups": "qsfs groups",
      "compression": "qsfs compression"
    }
  }
}
```

- `v4/zos.project.qsfs.get`  (project name, qsfs date)

```json
{
  "command": "v4/zos.project.qsfs.get",
  "data": {
    "project_name": "project name",
    "name": "qsfs name",
  },
  "result": {
    "status": "deleted", 
    "error": ""
  }
}
```

### ZLog

- `v4/zos.project.zlog.create`  (project name, zlog date)

```json
{
  "command": "v4/zos.project.zlog.create",
  "data": {
    "project_name": "project name",
    "name": "zlog name",
    "version": "deployment version",
    "vm_name": "vm name",
    "output": "zlog output"
  },
  "result": {
    "status": "created", 
    "error": ""
  }
}
```

- `v4/zos.project.zlog.update`  (project name, zlog date)

```json
{
  "command": "v4/zos.project.zlog.update",
  "data": {
    "project_name": "project name",
    "name": "zlog name",
    "version": "deployment version",
    "vm_name": "vm name",
    "output": "zlog output"
  },
  "result": {
    "status": "ok", 
    "error": ""
  }
}
```

- `v4/zos.project.zlog.get`  (project name, zlog date)

```json
{
  "command": "v4/zos.project.zlog.get",
  "data": {
    "project_name": "project name",
    "name": "zlog name"
  },
  "result": {
    "status": "ok", 
    "error": "",
    "data": {
      "project_name": "project name",
      "name": "zlog name",
      "version": "deployment version",
      "vm_name": "vm name",
      "output": "zlog output"
    }
  }
}
```

- `v4/zos.project.zlog.delete`  (project name, zlog date)

```json
{
  "command": "v4/zos.project.zlog.delete",
  "data": {
    "project_name": "project name",
    "name": "zlog name"
  },
  "result": {
    "status": "deleted", 
    "error": "",
    "data": {}
  }
}
```

### gateway FQDN

- `v4/zos.project.gateway.fqdn.create`  (project name, gateway date)

```json
{
  "command": "v4/zos.project.gateway.fqdn.create",
  "data": {
    "project_name": "project name",
    "name": "gateway name",
    "version": "deployment version",
    "fqdn": "fqdn",
    "tls_passthrough": "tls passthrough is optional",
    "network": "gateway network",
    "backends": ["list of backends"]
  },
  "result": {
    "status": "created", 
    "error": ""
  }
}
```

- `v4/zos.project.gateway.fqdn.update`  (project name, gateway date)

```json
{
  "command": "v4/zos.project.gateway.fqdn.update",
  "data": {
    "project_name": "project name",
    "name": "gateway name",
    "version": "deployment version",
    "fqdn": "fqdn",
    "tls_passthrough": "tls passthrough is optional",
    "network": "gateway network",
    "backends": ["list of backends"]
  },
  "result": {
    "status": "ok", 
    "error": ""
  }
}
```

- `v4/zos.project.gateway.fqdn.get`  (project name, gateway date)

```json
{
  "command": "v4/zos.project.gateway.fqdn.get",
  "data": {
    "project_name": "project name",
    "name": "gateway name"
  },
  "result": {
    "status": "ok", 
    "error": "",
    "data": {
      "project_name": "project name",
      "name": "gateway name",
      "version": "deployment version",
      "fqdn": "fqdn",
      "tls_passthrough": "tls passthrough is optional",
      "network": "gateway network",
      "backends": ["list of backends"]
    }
  }
}
```

- `v4/zos.project.gateway.fqdn.delete`  (project name, gateway date)

```json
{
  "command": "v4/zos.project.gateway.fqdn.get",
  "data": {
    "project_name": "project name",
    "name": "gateway name"
  },
  "result": {
    "status": "deleted", 
    "error": "",
    "data": {}
  }
}
```

### gateway name

- `v4/zos.project.gateway.name.create`  (project name, gateway date)

```json
{
  "command": "v4/zos.project.gateway.name.create",
  "data": {
    "project_name": "project name",
    "name": "gateway name",
    "version": "deployment version",
    "tls_passthrough": "tls passthrough is optional",
    "network": "gateway network",
    "backends": ["list of backends"]
  },
  "result": {
    "status": "created", 
    "error": ""
  }
}
```

- `v4/zos.project.gateway.name.update`  (project name, gateway date)

```json
{
  "command": "v4/zos.project.gateway.name.update",
  "data": {
    "project_name": "project name",
    "name": "gateway name",
    "version": "deployment version",
    "tls_passthrough": "tls passthrough is optional",
    "network": "gateway network",
    "backends": ["list of backends"]
  },
  "result": {
    "status": "ok", 
    "error": ""
  }
}
```

- `v4/zos.project.gateway.name.get`  (project name, gateway date)

```json
{
  "command": "v4/zos.project.gateway.name.get",
  "data": {
    "project_name": "project name",
    "name": "gateway name"
  },
  "result": {
    "status": "ok", 
    "error": "",
    "data": {
      "project_name": "project name",
      "name": "gateway name",
      "version": "deployment version",
      "tls_passthrough": "tls passthrough is optional",
      "network": "gateway network",
      "backends": ["list of backends"]
    }
  }
}
```

- `v4/zos.project.gateway.name.delete`  (project name, gateway date)

```json
{
  "command": "v4/zos.project.gateway.name.get",
  "data": {
    "project_name": "project name",
    "name": "gateway name"
  },
  "result": {
    "status": "deleted", 
    "error": "",
    "data": {}
  }
}
```

### public IP

- `v4/zos.project.public_ip.create`  (project name, public ip date)

```json
{
  "command": "v4/zos.project.public_ip.create",
  "data": {
    "project_name": "project name",
    "name": "public_ip name",
    "version": "deployment version",
    "ipv4": "if you want an ipv4",
    "ipv6": "if you want an ipv6"
  },
  "result": {
    "status": "created", 
    "error": ""
  }
}
```

- `v4/zos.project.public_ip.update`  (project name, public ip date)

```json
{
  "command": "v4/zos.project.public_ip.update",
  "data": {
    "project_name": "project name",
    "name": "public_ip name",
    "version": "deployment version",
    "ipv4": "if you want an ipv4",
    "ipv6": "if you want an ipv6"
  },
  "result": {
    "status": "ok", 
    "error": ""
  }
}
```

- `v4/zos.project.public_ip.get`  (project name, public ip date)

```json
{
  "command": "v4/zos.project.public_ip.get",
  "data": {
    "project_name": "project name",
    "name": "public_ip name"
  },
  "result": {
    "status": "ok", 
    "error": "",
    "data": {
      "project_name": "project name",
      "name": "public_ip name",
      "version": "deployment version",
      "ipv4": "if you want an ipv4",
      "ipv6": "if you want an ipv6"
    }
  }
}
```

- `v4/zos.project.public_ip.delete`  (project name, public ip date)

```json
{
  "command": "v4/zos.project.public_ip.delete",
  "data": {
    "project_name": "project name",
    "name": "public_ip name"
  },
  "result": {
    "status": "deleted", 
    "error": "",
    "data": {}
  }
}
```
