#!/bin/bash
cat gon.hcl | sed -r s/\{\{ artifact \}\}/$1/ | sed -r s/\{\{ username \}\}/$USERNAME/ | sed -r s/\{\{ password \}\}/$PASSWORD/ > gon_processed.hcl
gon --log-level=DEBUG gon_processed.hcl
