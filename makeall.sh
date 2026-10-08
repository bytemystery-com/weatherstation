#!/bin/bash

GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" .
sudo systemctl stop weatherd.service
sudo cp weatherd /usr/local/bin
sudo systemctl start weatherd.service

