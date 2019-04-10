# DSL 
DSL is the *domain specific language* used to communicate with a single zero-os node. The language is designed to 
allocate and build resources with great flexibility. The DSL language is like an instruction blueprint where user
can use it to pull different type of primitives together to build a working solution.

# Content-Type
The API is probably going to support multiple content-types, json, yaml, etc... in the rest of this document we going to use `yaml` to
show structures, and objects for readability.

# Philosophy
Here we gonna list what the DSL is all about and how it should behave. This is an open discussion area. This is basically the ideas on 
how the system should interface with the world. Hence it's more of the philosophy of zos, rather than the specs. The following rules is
going to define how the entire system should behave.

- The main entity of the API is a `resource`, you can create, list, inspect and destroy a resource
- The DSL is a language used to describe a resource on creation, and returned on inspection
- A resource `create` operation can create a single resource, a single resource can be a single or multiple allocations
- An allocation limited to storage, network, memory, and cpu.
- A resource `create` must be atomic. in other words all allocations must pass before the resource is considered created.
- A resource must have a unique id, and any set of searchable tags for fast find.
- Each allocation, must have unique name (in the same resource)
- In a single resource, allocations can have dependencies, and can reference each others

General resource structure
```yaml
id: <id>  # optional: if left empty a unique one will be generated
tags:
    name: <name>
    custom: <custom>
    # tags are indexed and should be searchable, they are defined by the client.
allocations:
    - type: <allocation type> # one of pre defined set of allocations types (for example storage, cpu, memory, flist, etc...)
      name: <allocation name> # for referencing inside the same resource, must be unique
      options:
        arg1: val1
        arg2: val2 
        # options, depends on the allocation type, this should be documented in the zos and api docs
        # please check examples below for how this going to be used
    - type: <allocation type>
      name: <allocation name>
```

## Examples
The example section is to show some use cases of how the DSL will look like

### ZDB Instance
A public ZDB instance resource

```yaml
tags:
    name: my-zdb
    type: zdb 
    # again, tags are defined by the client, a find query can use one or multiple tags to list
    # *all* resources with the given set of tags. If direct access get the ID
allocations:
    - type: storage
      name: disk
      options:
         size: 10G
         type: ssd
    - type: port # type port, reserves the first available free port
      name: port      
    - type: zdb
      name: zdb
      options:
         storage: @disk # we use @ to refer to disk, this way we know disk allocation needs to be created first
         port: @port # the @ reference the name of the allocation of name `port` this can be also given as `uint16` value
    
```

### Container
```yaml
allocations:
    - type: flist
      name: root
      options:
        flist: https://hub.grid.tf/thabet/redis.flist
        size: 5G # max space allocated to `write` layer
    - type: storage
      name: data
      options:
        size: 100G
        type: hdd
    - type: network
      name: default
      options:
        # to be defined
    - type: network
      name: vxlan
      options:
        vxlan: 123
    - type: port
      name: redis
    - type: container
      name: container
      options:
        memory: 1024M
        cpu: 2
        root: @root
        mounts:
            - @data: /var/data
        ports:
            - @redis: 6379
        env:
            KEY: VALUE
        entry-point: redis-server 
        # more options in container allocation
```

### User Space
A user space is an isolated space owned by a customer/user where a user can then do more sub allocations on, to create containers or VMs

```yaml
tags:
    user: user-id
allocations:
    - type: storage
      name: ssd-storage
      options:
        size: 100G
        type: ssd
    - type: storage
      name: hdd-storage
      options:
        size: 100G
        type: hdd
    - type: cpu
      name: cpu
      options:
        cores: 8
    - type: memory
      name: memory
      options:
        size: 128G
    - type: port
      name: api-port
    - type: space
      name: space
      options:
        port: @api-port
        storage:
          - @ssd-storage
          - @hdd-storage
        cpu: @cpu
        memory: @memory
        network: <network> # we need discuss network allocation for space
        user:
          - public.key: |
            ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDXKwi6QhxCb/Ep7Kp+3vkffRVcS9OIJAIuk3TS/fNkRU9lXijlvThuV4+hkz2ZDtK5D+DBHwRrNj8SY3b9X1WC/Xhh3pQl9RlDld+459c966iqOrdLjchnRqiQ6fQXwPA0rJqa5suKGMoGFdJDcNtiIkf3Ht0hF6Hps/EMaDxkVAUvaIS5uqg/iNVUK9x5rFOd3Y2KDtu0PTiPQ5zNGOhmhLOy1QQ1kDraIuvb3tJR7c9Y8H4WyB42j6nG/m8ZdHfnMwLp5ERTkRfZLF5sBit7gBfSCNVgFH4d7zEQzY1FtBPzqg15cgt7eVhIcwn9A6TojfCQnxv6m2VZ22oxlOxn azmy@curiosity
```