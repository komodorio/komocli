#!/bin/bash

echo \$1 | base64 --decode >certificate.p12
sudo security create-keychain -p actions build.keychain
sudo security default-keychain -s build.keychain
sudo security unlock-keychain -p actions build.keychain
sudo security import certificate.p12 -k build.keychain -P "" -T /usr/bin/codesign
sudo security set-key-partition-list -S apple-tool:,apple: -s -k actions build.keychain
rm certificate.p12
