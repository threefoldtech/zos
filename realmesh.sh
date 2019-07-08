#!/usr/bin/env bash

# create a set of configfiles to mesh publically reachable
# nodes with Wireguard 

list=( "[2a03:b0c0:1:d0::11:c001],TF-dns-03-DO-lon1"
       "[2a03:b0c0:3:e0::2ce:d001],TF-dns-02-DO-fra1"
       "[2a03:b0c0:2:d0::2fd:1001],TF-dns-01-DO-ams3"
       "[2a03:b0c0:2:d0::189:3001],WebSites-01-ams3"
       "[2a03:b0c0:2:d0::93:9001],WebSites-02-ams3"
     )

# setup 2 network namespaces, generate keys for 2 wg's

function genkeys() {
  for i in ${list[@]}; do
    wgname=wg-${i##*,}
    wg genkey | tee ${wgname}.priv | wg pubkey >${wgname}.pub
  done
}

function genconf() {
  localip=1
  for i in ${list[@]}; do
    wgname=wg-${i##*,}
    echo "${wgname}.."
    PRIV=$(cat ${wgname}.priv)
    port=16000
    cat <<EOF >${wgname}.conf
# ${wgname}
[Interface]
ListenPort = ${port}
PrivateKey = $PRIV
Address = 192.168.255.${localip}/24
EOF
    cnt=1
    for wg in ${list[@]}; do
      hcnt=$(printf '%x' $cnt)
      peer=wg-${wg##*,}
      if [ ! "${peer}" = "$wgname" ]; then
        port=16000
        PUB=$(cat ${peer}.pub)
        PEERIP=${wg%,*}
        cat <<EEOF >>${wgname}.conf

# Config for --- ${peer} ---
[Peer]
PublicKey = $PUB
Endpoint = ${PEERIP}:${port}
AllowedIPs = fe80::${hcnt},192.168.255.${cnt},2001:1:1:${hcnt}::/64
PersistentKeepalive = 20
EEOF
      fi
      let cnt++
    done
      let localip++
  done
  echo
}

# watch out! wireguard needs to be installed on the node **and**
# the wg-quick tool needs to be available
# also : no errcheck whatsoever, watch your step
function wginstall(){
  for i in ${list[@]}; do
    config=wg-${i##*,}.conf
    dest=${i%,*}
    scp ${config} root@${dest}:/etc/wireguard/wg0.conf
    ssh root@${dest} "wg-quick down wg0; wg-quick up wg0"

  done
}