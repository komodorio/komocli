#!/bin/bash
export ARTIFACT=$1
envsubst < gon.hcl > gon_processed.hcl
gon --log-level=DEBUG gon_processed2.hcl
