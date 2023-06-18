#!/bin/sh

# Decode the certificate
echo $MACOS_CERTIFICATE_P12 >certificate.p12

# Create a keychain
# security create-keychain -p actions build.keychain

# Import the certificate
# security import certificate.p12 -k build.keychain -P "" -T /usr/bin/codesign

# Remember the keychain
# security list-keychains -s build.keychain

# Unlock the keychain
# security unlock-keychain -p actions build.keychain

# Find the macOS binary and sign it
# TODO "$CERTIFICATE_ID"
zsign/build/zsign --pkey certificate.p12 $1
