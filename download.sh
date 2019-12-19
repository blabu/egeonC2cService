#!/bin/bash

binURL=http://localhost:6060/upload/bin
confURL=http://localhost:6060/upload/conf

rm -rf ./test
mkdir ./test
mount -t tmpfs -o size=16m tmpfs ./test
cd ./test
echo "Try download binary data from " $binURL
wget -O c2cService $binURL
chmod u+x ./c2cService
wget -O config.conf $confURL
./c2cService
