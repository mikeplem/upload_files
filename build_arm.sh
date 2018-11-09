#!/bin/bash
env GOOS=linux GOARCH=arm CGO_ENABLED=0 go build -o uploadvideo_arm main.go ldap.go
