nft 'add table inet filter'
nft 'add table arp filter'
nft 'add table bridge filter'

# duo to a bug we had we need to make sure those chains are
# deleted and then recreated later
nft 'delete chain inet filter input'
nft 'delete chain inet filter forward'
nft 'delete chain inet filter output'

nft 'delete chain bridge filter input'
nft 'delete chain bridge filter forward'
nft 'delete chain bridge filter output'

nft 'delete chain arp filter input'
nft 'delete chain arp filter output'

# recreate chains correctly
nft 'add chain inet filter input   { type filter hook input priority filter; policy accept; }'
nft 'add chain inet filter forward { type filter hook forward priority filter; policy accept; }'
nft 'add chain inet filter output  { type filter hook output priority filter; policy accept; }'
nft 'add chain inet filter prerouting  { type filter hook prerouting priority filter; policy accept; }'

nft 'add chain arp filter input  { type filter hook input priority filter; policy accept; }'
nft 'add chain arp filter output { type filter hook output priority filter; policy accept; }'

nft 'add chain bridge filter input   { type filter hook input priority filter; policy accept; }'
nft 'add chain bridge filter forward { type filter hook forward priority filter; policy accept; }'
nft 'add chain bridge filter prerouting { type filter hook prerouting priority filter; policy accept; }'
nft 'add chain bridge filter postrouting { type filter hook postrouting priority filter; policy accept; }'
nft 'add chain bridge filter output  { type filter hook output priority filter; policy accept; }'

nft 'flush chain bridge filter forward'
nft 'flush chain inet filter forward'
nft 'flush chain inet filter prerouting'

# drop smtp traffic for hidden nodes
nft 'add rule inet filter prerouting iifname "b-*" tcp dport {25, 587, 465} reject with icmp type admin-prohibited'

# prevent access to local network
#nft 'add rule bridge filter output oif "{{ .Inf }}" ether daddr != "{{ .Gateway }}" drop'
