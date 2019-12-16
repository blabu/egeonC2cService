#!/bin/bash

binURL=http://localhost:6060/bin
confURL=http://localhost:6060/bin/conf

rm -rf ./test
mkdir ./test
cd ./test
echo "Try download binary data from " $binURL
wget -O c2cService $binURL
chmod u+x ./c2cService
wget -O config.conf $confURL
./c2cService
