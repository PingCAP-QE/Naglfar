#!/bin/bash

procs=$(for d in $(ps aux | grep "$PROC" | grep -v grep | tail -n+1 | awk '{print "" "\""$2"\"" ","}'); do echo $d; done | sed '$s/.$//')

echo "[$procs]"
