#!/bin/bash

ROOT_FOLDER="/home/reiner/SSD_1T/weatherstation"
RSYNC_IMG="weather-rsync.img"
RSYNC_MOUNT="weather_rsync"
IP="192.168.0.130"
IMG_NAME="weather-rync_$(date +%Y%m%d).img"
PISHRINK="${HOME}/bin/pishrink.sh"

set -eo pipefail

function cleanup()
{
	sudo umount ${LOOP}p1 || true
	sudo umount ${LOOP}p2 || true
	sudo losetup -d ${LOOP} || true
}

trap cleanup EXIT

mkdir -p "${ROOT_FOLDER}/${RSYNC_MOUNT}"

LOOP=$(sudo losetup -f --show --partscan "${ROOT_FOLDER}/${RSYNC_IMG}")
sudo mount ${LOOP}p2 "${ROOT_FOLDER}/${RSYNC_MOUNT}"
sudo mount ${LOOP}p1 "${ROOT_FOLDER}/${RSYNC_MOUNT}/boot/firmware"

sudo rsync -av --delete --numeric-ids --exclude={'/dev/*','/proc/*','/sys/*','/tmp/*','/run/*','/mnt/*','/media/*','/lost+found'} root@${IP}:/ "${ROOT_FOLDER}/${RSYNC_MOUNT}"
cleanup
sudo rm -f "${ROOT_FOLDER}/${IMG_NAME}.xz" || true
sudo ${PISHRINK} -rZ "${ROOT_FOLDER}/${RSYNC_IMG}" "${ROOT_FOLDER}/${IMG_NAME}.xz"



