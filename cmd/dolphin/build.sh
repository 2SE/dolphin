#! /bin/bash

#CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o myapp
docker build -t 192.168.9.130:5000/hashhash/dolphin:$1 .
#rm -rf myapp

