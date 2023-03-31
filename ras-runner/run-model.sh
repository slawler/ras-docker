#!/usr/bin/bash

# usage: ./run-model.sh "Muncie" "04" "04"

set -euo pipefail

MODELNAME=$1
GEOM_ID=$2
UNSTEADY_ID=$3

RAS_LIB_PATH=/ras/libs:/ras/libs/mkl:/ras/libs/rhel_8

if [[ -v LD_LIBRARY_PATH ]]; then
    # env var already exists, so append to it
    export LD_LIBRARY_PATH=$RAS_LIB_PATH:$LD_LIBRARY_PATH
else
    # env var does not already exist, so cannot append to it
    export LD_LIBRARY_PATH=$RAS_LIB_PATH
fi

RAS_EXE_PATH=/ras/v61
export PATH=$RAS_EXE_PATH:$PATH

RasUnsteady "${MODELNAME}.c${GEOM_ID}" "b${UNSTEADY_ID}"
