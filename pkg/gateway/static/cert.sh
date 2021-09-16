DOMAIN="$2"
DOMAIN=${DOMAIN#"*."}
TOKEN="$3"
IP=`ip -4 addr show public | grep inet | tr -s ' ' | cut -d' ' -f3 | cut -d'/' -f1`
if [ -z "$IP" ]; then
    echo no public ip found
    exit 1
fi

if [ $1 == "present" ]; then
    pkill dnsmasq
    cp %[1]s/dnsmasq.conf /tmp/dnsmasq.conf
    sed -i "s/DOMAIN/$DOMAIN/g" /tmp/dnsmasq.conf
    sed -i "s/TOKEN/$TOKEN/g" /tmp/dnsmasq.conf
    sed -i "s/IP/$IP/g" /tmp/dnsmasq.conf
    dnsmasq -C /tmp/dnsmasq.conf
elif [ $1 == "timeout" ]; then
    echo '{"timeout": 30, "interval": 5}'
else
    pkill dnsmasq
    rm -rf /tmp/dnsmasq
fi