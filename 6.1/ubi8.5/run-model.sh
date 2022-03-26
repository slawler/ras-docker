#!/usr/bin/sh

# usage ./run.sh /sim/sample-model/ Muncie

MODELDIR=$1
MODEL=$2

RAS_LIB_PATH=/ras/libs:/ras/libs/mkl:/ras/libs/rhel_8
export LD_LIBRARY_PATH=$RAS_LIB_PATH:$LD_LIBRARY_PATH

RAS_EXE_PATH=/ras/v61
export PATH=$RAS_EXE_PATH:$PATH

cd $MODELDIR
RasUnsteady $2.c04 b04