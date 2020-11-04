#!/bin/bash

# Debugging and error handling
# this is a good option.
set -e
# Debugging options:
# set -x
# set -v


# General OS props
HOST=$HOSTNAME
ARCHITECTURE=$(uname -m)


# Memory
memTotal=$(egrep '^MemTotal:' /proc/meminfo | awk '{print $2}')

# CPU
cpuThreads=$(grep processor /proc/cpuinfo | wc -l)

# Disk
disksJson=$(for d in $(df -P -x tmpfs -x devtmpfs -x ecryptfs -x nfs -x cifs -T \
| tail -n+2 \
| awk '{print "" "\""$1"\"" ": {\"filesystem\":" "\""$2"\"" ", \"total\":" "\""$3"KiB\"" ", \"used\":" "\""$4"KiB\"" ", \"mountPoint\":" "\""$7"\"" "},"}'); \
do echo $d; done | sed '$s/.$//')

# Final result in JSON
JSON="
{
  \"hostname\": \"$HOST\",
  \"architecture\": \"$ARCHITECTURE\",
  \"memory\": \""$memTotal"KiB\",
  \"threads\": $cpuThreads,
  \"devices\": {
    $disksJson
  }
}"

echo "$JSON"
