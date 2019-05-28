#!/bin/bash

echo "Enable ip forwarding"
echo 1 > /proc/sys/net/ipv4/ip_forward