#!/usr/bin/env bash

# setup connection namespaces with a dummy and add an ip
NUM=5

function prepare(){
	# mock NR (Network REsource) IPv4
	for i in $(seq 1 $NUM) ; do
		ip netns add z${i}
		ip -n z${i} link set lo up
		ip -n z${i} link add zone${i} type dummy
		ip -n z${i} link set zone${i} up
		ip -n z${i} addr add 10.10.0.1/24 dev zone${i}
	done

	# Namespace for getting IPv4 NATed
	ip netns add vrf

	# Ultimate exit: ovrf has penultimate route
	ip link add ivrf type veth peer name ovrf
	ip link set ivrf netns vrf
	ip -n vrf link set ivrf up
	ip -n vrf link set lo up
	ip -n vrf addr add 172.18.0.254/24 dev ivrf
	ip -n vrf link add cvrf type dummy

	# connect NRs to vrf instance
	for i in $(seq 1 $NUM) ; do
		ip link add oz${i} type veth peer name iz${i}
		ip link set iz${i} netns z${i}
		ip -n z${i} link set iz${i} up
		ip -n z${i} addr add 172.16.0.1/24 dev iz${i}
		ip -n z${i} route add default via 172.16.0.254
		ip link set oz${i} netns vrf
		ip -n vrf link set oz${i} up
	done

	ip link set ovrf up

	# setup vrf
	for i in $(seq 1 $NUM) ; do
		ip link add vz${i} type vrf table ${i}
		ip link set oz${i} master vz${i}
		ip addr add 172.16.0.254/24 dev oz${i}
		ip link set vz${i} up
		ip linl set oz${I} up
	done

}

function delete(){
	for i in $(seq 1 $NUM) ; do
		ip netns del z${i}
		ip link del oz${i}
	done
	ip netns del vrf
	ip link del ovrf

}

