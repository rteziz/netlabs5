#!/bin/sh
set -eu

rate="${RATE:-80mbit}"
if [ "$rate" != 0 ]; then
    # Limit incoming file data, not just the server's outgoing ACKs.
    tc qdisc add dev eth0 handle ffff: ingress
    tc filter add dev eth0 parent ffff: protocol all u32 \
        match u32 0 0 \
        action police rate "$rate" burst 256k conform-exceed drop/ok
    echo "Incoming traffic limit: $rate (shared by all clients)"
else
    echo "Network rate limit disabled"
fi

exec /usr/local/bin/server "$@"
