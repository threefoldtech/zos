#!/usr/bin/bash

# setup NUM network namespaces that are connecte via a circular mesh:
# i.e. every namespace has an encrypted tunnel to the other, with the
# associated route

NUM=50
function genkeys() {
	for i in $(seq 1 $NUM); do
		wg genkey | tee wg${i}.priv | wg pubkey >wg${i}.pub
	done
}

function genconf() {
	for i in $(seq 1 $NUM); do
		echo -n "${i}.."
		PRIV=$(cat wg${i}.priv)
		h=$(printf '%x' $i)
		port=$((16000 + $i))
		cat <<EOF >wg${i}.conf
# WG${i}
[Interface]
ListenPort = 1234${i}
PrivateKey = $PRIV
EOF
		for wg in $(seq 1 $NUM); do
			if [ "$wg" -ne "$i" ]; then
				PUB=$(cat wg${wg}.pub)
				cat <<EEOF >>wg${i}.conf

# Config for --- WG${wg} ---
[Peer]
PublicKey = $PUB
Endpoint = 127.0.0.1:${port}
AllowedIPs = fe80::${h},192.168.255.${wg},2001:1:1:${h}::/64
PersistentKeepalive = 20
EEOF
			fi
		done
	done
	echo 
}

function ns() {
	for i in $(seq 1 $NUM); do
		echo -n "${i}.."
		h=$(printf '%x' $i)
		ip netns add wg${i}

		ip link add wg${i} type wireguard
		ip link set wg${i} netns wg${i}

		ip link add int${i} type dummy
		ip link set int${i} netns wg${i}

		ip -n wg${i} link set lo up
		ip -n wg${i} link set wg${i} up
		ip -n wg${i} link set int${i} up

		ip netns exec wg${i} wg setconf wg${i} wg${i}.conf

		ip -n wg${i} addr add fe80::${h}/64 dev wg${i}
		ip -n wg${i} addr add 192.168.255.${i}/24 dev wg${i}

		ip -n wg${i} addr add 2001:1:1:${h}::1/64 dev int${i}
	done
	echo
}

function addroutes() {
	
	for i in $(seq 1 $NUM); do
		echo -n "${i}.."
		for wg in $(seq 1 $NUM); do
			if [ "$wg" -ne "$i" ]; then
				h=$(printf '%x' $wg)
				# echo ip -n wg${i} route add 2001:1:1:${h}::/64 via fe80::${h} dev wg${i}
				ip -n wg${i} route add 2001:1:1:${h}::/64 via fe80::${h} dev wg${i}
			fi
		done
		# default route is via wg1
		[ "$i" -ne "1" ] && ip -n wg${i} route add default via fe80::1 dev wg${i}
	done

}

function main() {
	genkeys
	echo
	genconf
	echo
	ns
	echo
	addroutes
	echo
}

# main
