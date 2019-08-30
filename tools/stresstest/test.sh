#!/bin/bash
set -ex

tfubin="${PWD}/../tfuser/tfuser"
schemas="${PWD}/schemas"

farmid="CemYjciEmuvYVKDFXYaZLdGsCdLDRp4U1Xu1LPPrQNkK"
tnodb="https://tnodb.dev.grid.tf"

redislog="10.4.0.250"
redischan="debug-$(date +%s)"

echo "[+] setting up environment"
rm -rf ${schemas}
mkdir -p ${schemas}

echo "[+] generating identity"
$tfubin id -o ${schemas}/user.seed
identity=$($tfubin id -o ${schemas}/user.seed | grep 'identity:' | awk '{ print $2 }')

echo "[+] identity: ${identity}"

echo "[+] fetching farms nodes list"
nodesjson=$(curl -s "${tnodb}/nodes?farm=${farmid}" | python -m json.tool)

echo "[+] selecting one exit node"
node=$(echo "$nodesjson" | egrep 'exit_node|node_id' | grep -E -B1 ': [1-9]' | head -1 | awk -F'"' '{ print $4 }')

echo "[+] exit node selected: $node"

echo "[+] creating a new network"
$tfubin generate network create --node $node > ${schemas}/net-init.json
netid=$(cat ${schemas}/net-init.json | python -m json.tool | grep network_id | awk -F'"' '{ print $4 }')

echo "[+] adding user to network"
wgkey=$($tfubin generate --schema ${schemas}/net-init.json network add-user --user ${identity} | head -1 | awk '{ print $4 }')

echo "[+] generating wireguard config"
$tfubin generate --schema ${schemas}/net-init.json network wg --user ${identity} --key ${wgkey} > ${schemas}/wg.conf

echo "[+] generating debug mode (redis ${redislog} -> ${redischan})"
$tfubin generate debug --endpoint "${redislog}:6379" --channel ${redischan} > ${schemas}/debug-node.json

echo "[+] generating container"
$tfubin container --flist https://hub.grid.tf/maxux/busybox-latest.flist --entrypoint /bin/ash --corex --network ${netid} --envs hello=world > ${schema}/busybox-corex.json

echo "[+] generating zdb profiles"
$tfubin generate storage zdb --size 10 --type SSD --mode user > ${schemas}/zdb-ssd-10.json
$tfubin generate storage zdb --size 100 --type SSD --mode user > ${schemas}/zdb-ssd-100.json
$tfubin generate storage zdb --size 500 --type SSD --mode user > ${schemas}/zdb-ssd-500.json

$tfubin generate storage zdb --size 100 --type SSD --mode user > ${schemas}/zdb-hdd-100.json
$tfubin generate storage zdb --size 1000 --type HDD --mode user > ${schemas}/zdb-hdd-1000.json

echo "[+]"
echo "[+] sending provisioning"
echo "[+]"

seed="${schemas}/user.seed"
duration="1h"

echo "[+] sending debug provisioning"
$tfubin provision --node ${node} --duration ${duration} --seed ${seed} --schema ${schemas}/debug-node.json

echo "[+] provisioning network"
$tfubin provision --node ${node} --duration ${duration} --seed ${seed} --schema ${schemas}/net-init.json

echo "[+] provisioning container"
$tfubin provision --node ${node} --duration ${duration} --seed ${seed} --schema ${schemas}/busybox-corex.json

echo "[+] provisioning zdb"
$tfubin provision --node ${node} --duration ${duration} --seed ${seed} --schema ${schemas}/zdb-ssd-10.json
$tfubin provision --node ${node} --duration ${duration} --seed ${seed} --schema ${schemas}/zdb-ssd-100.json
$tfubin provision --node ${node} --duration ${duration} --seed ${seed} --schema ${schemas}/zdb-ssd-500.json
$tfubin provision --node ${node} --duration ${duration} --seed ${seed} --schema ${schemas}/zdb-hdd-100.json
$tfubin provision --node ${node} --duration ${duration} --seed ${seed} --schema ${schemas}/zdb-hdd-1000.json

