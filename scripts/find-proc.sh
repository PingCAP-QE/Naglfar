#!/bin/bash

procs=$(for d in $(ps aux | grep "$PROC" | grep -v grep | tail -n+2 | awk '{print "" "\""$2"\"" ","}'); do echo $d; done | sed '$s/.$//')
# do echo $d; done | sed '$s/.$//')

echo "[$procs]"
