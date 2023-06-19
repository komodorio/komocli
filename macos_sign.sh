#!/bin/bash
cat gon.hcl | sed "s~=artifact=~$1~g" | sed "s/=username=/$USERNAME/g" | sed "s/=password=/$PASSWORD/g" > gon_processed.hcl
gon --log-level=DEBUG gon_processed.hcl
