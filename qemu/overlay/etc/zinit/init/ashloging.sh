#!/bin/bash

echo "start ash terminal"
# Enable ip forwarding
while true; do
getty -l /bin/ash -n 19200 ttyS1
done
