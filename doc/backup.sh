#!/bin/bash

set -euo pipefail

USER='psyst'
IP='192.168.0.130'
FOLDER="${HOME}/SSD_1T/weatherstation"
IMG="${FOLDER}/weather_$(date +%Y%m%d).img"

ssh "${USER}@${IP}" "sudo dd if=/dev/mmcblk0 bs=4M status=progress" | dd of="${IMG}" bs=4M
sudo pisshrink.sh -raZ "${IMG}"
