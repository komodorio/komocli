#!/usr/bin/env sh
echo "Signing $1"
codesign --options runtime --keychain build.keychain --sign "$CERTIFICATE_ID" $1 2>&1 | tee -a codesign.log
