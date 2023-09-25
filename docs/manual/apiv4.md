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
  "name": "project name",
  "twin_id": "user twin id",
  "contracts_id": "project contracts id",
  "metadata": "project metadata",
  "description": "project description",
  "expiration": "project expiration",
  "signature": "user signature"
 }
}
```

- `v4/zos.project.list` -> lists all projects for the user that made the calls

```json
{
 "command": "v4/zos.project.list",
 "data": {},
 "result": [{
  "name": "project name",
  "twin_id": "user twin id",
  "contracts_id": "project contracts id",
  "metadata": "project metadata",
  "description": "project description",
  "expiration": "project expiration",
  "signature": "user signature"
 }, ...]
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

- `v4/zos.project.network.create`  (project name, component date)

```json
{
 "command": "v4/zos.project.network.create",
 "data": {
  "project_name": "project name"
 ""
 },
 "result": {
  "name": "project name",
  "twin_id": "user twin id",
  "contracts_id": "project contracts id",
  "metadata": "project metadata",
  "description": "project description",
  "expiration": "project expiration",
  "signature": "user signature"
 }
}
```

- `v4/zos.project.network.restart`  (project name, component date)

- `v4/zos.project.network.get`   (project name, component name)

- `v4/zos.project.network.update`  (project name, component data)

- `v4/zos.project.network.delete`   (project name, component name)

- VM (fails if no network is created for the same project)
- disk
- zdb
- qsfs
- zlog
- zmount
- fqdn gateway
- name gateway
- public ipv4

### Component data

- Version
- Name
- Project name
- Metadata
- Description
- Data
- Result
