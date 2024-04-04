#!/usr/bin/env sh
echo "Notarizing $1"
zip $1.zip $1
xcrun notarytool submit $1.zip \
    --apple-id "$MACOS_NOTARY_APPLE_ID" \
    --team-id "$MACOS_NOTARY_TEAM_ID" \
    --password "$MACOS_NOTARY_PASSWORD" \
    --wait 2>&1 | tee -a notarytool.log
