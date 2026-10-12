#!/bin/bash

CLEAN=0

go fmt ./...    
go vet ./... 
go get -u ./... # ???   
go mod tidy
if [[ ${CLEAN} -eq 1 ]] ; then
    go clean -cache -modcache
fi

command -v cloc >/dev/null 2>&1
if [[ $? -eq 0 ]] ; then
    cloc .
fi


GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" .
sudo systemctl stop weatherd.service
sudo cp weatherd /usr/local/bin
sudo systemctl start weatherd.service

