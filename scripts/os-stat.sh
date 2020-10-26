#!/bin/bash

# Collects system performance statistics such as CPU, memory, and disk
# usage as well as top processes ran by users.
#
# All size values are in KiB (memory, disk, etc).


# Takes these command line arguments:
# $1 - cpuThreshold in % of total across all CPUs. Default is provided in no-args option.
# $2 - memThreshold in % of total avail. Default is provided in no-args option.
#
# EXAMPLE USAGE:
# ./os_stats.sh 5 20


# Debugging and error handling
# Stop this script on any error. Unless you want to get data by any means
# this is a good option.
set -e
# Debugging options:
# set -x
# set -v


# Validate command line arguments
if [[ "$#" == 1 || "$#" > 2 ]]; then
  echo "Wrong number of arguments supplied. Expects 2 arguments: <cpu>, <mem>, or none."
  exit 1
fi


# Configurable variables
#
# CPU threshold in %of total CPU consumption (100%)
# All processes below this threshold will be discarded.
if [ -z "$1" ]; then
  cpuThreshold="1"
else
  cpuThreshold="$1"
fi
# % of memory usage. All processes below this threshold will be discarded.
if [ -z "$2" ]; then
  memThreshold="10"
else
  memThreshold="$2"
fi


# General OS props
HOST=$HOSTNAME
#OS=$(uname -a)
OS=$(lsb_release -s -i -c -r)
UPTIME=$(uptime)
ARCHITECTURE=$(uname -m)


# Memory
memTotal=$(egrep '^MemTotal:' /proc/meminfo | awk '{print $2}')
memFree=$(egrep '^MemFree:' /proc/meminfo | awk '{print $2}')
memCached=$(egrep '^Cached:' /proc/meminfo | awk '{print $2}')
memAvailable=$(expr "$memFree" + "$memCached")
#memUsed=$(($memTotal - $memAvailable))
memUsed=$(($memTotal - $memFree))
swapTotal=$(egrep '^SwapTotal:' /proc/meminfo | awk '{print $2}')
swapFree=$(egrep '^SwapFree:' /proc/meminfo | awk '{print $2}')
swapUsed=$(($swapTotal - $swapFree))


# CPU
cpuThreads=$(grep processor /proc/cpuinfo | wc -l)
#cpuUtilization=$(top -bn3 | grep "Cpu(s)" | sed "s/.*, *\([0-9.]*\)%* id.*/\1/" | awk '{print 100 - $1}' | tail -1)
cpuUtilization=$((100 - $(vmstat 2 2 | tail -1 | awk '{print $15}' | sed 's/%//')))


# Disk
disksJson=$(for d in $(df -P -x tmpfs -x devtmpfs -x ecryptfs -x nfs -x cifs -T | tail -n+2 | awk '{print "{" "\"total\":" $3 ", \"used\":" $4 ", \"mountPoint\":" "\""$7"\"" "},"}'); do echo $d; done | sed '$s/.$//')


# TODO: add cputime,start to list of ps columns
# Processes
curUser=""
processes=""
processesJson=$(ps --no-headers -eo user,pcpu,comm,pmem --sort=user | grep -v '<defunct>' | while read p; do
  arr=($p)
  user=${arr[0]}
  load=${arr[1]}
  command=${arr[2]}
  memory=${arr[3]}
  if [ "$curUser" != "$user" ]; then
    processes=$(echo "$processes" | sed '$s/.$//')
    if [[ -n "$curUser" && -n "$processes" ]]; then
      echo "{ \"username\": \"$curUser\", \"processes\": [ $processes ] },"
    fi
    curUser=$user
    processes=""
  fi
  load=$(echo "scale=2; $load / $cpuThreads" | bc -l | awk '{printf "%0.2f\n", $0}')
  if [[ "$load" > "$cpuThreshold" || "$memory" > "$memThreshold" ]]; then
    memUsedKb=$(echo "scale=0; $memory * $memTotal / 100 " | bc -l)
    processes+=" {\"cpuLoad\": $load, \"command\": \"$command\", \"memoryUsage\": $memUsedKb},"
  fi
done | sed '$s/.$//')


# Final result in JSON
JSON="
{
  \"hostname\": \"$HOST\",
  \"operatingSystem\": \"$OS\",
  \"uptime\": \"$UPTIME\",
  \"architecture\": \"$ARCHITECTURE\",
  \"memory\":
  {
    \"total\": $memTotal,
    \"used\": $memUsed,
    \"cache\": $memCached,
    \"swap\": $swapUsed
  },
  \"cpu\":
  {
    \"threads\": $cpuThreads,
    \"usedPercent\": $cpuUtilization
  },
  \"users\": [
    $processesJson
   ],
  \"disks\": [
    $disksJson
  ]
}"

echo "$JSON"

# Result output: STDOUT or HTTP
#if [ -z "$3" ]; then
#  echo "$JSON"
#else
#  curl -X POST -H "Content-Type: application/json" -d "$JSON" "$3"
#fi