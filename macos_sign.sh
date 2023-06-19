#!/bin/bash
export USERNAME="${{ secrets.APPLE_ID_USERNAME }}"
export PASSWORD="${{ secrets.APP_SPECIFIC_PASSWORD }}"
export ARTIFACT=$1
envsubst < gon.hcl > gon_processed.hcl
gon --log-level=DEBUG gon_processed2.hcl
