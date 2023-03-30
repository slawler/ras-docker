#!/bin/bash

set -euo pipefail

REGION=us-east-1
TAG=$1
ACCOUNT=$2.dkr.ecr.$REGION.amazonaws.com
IMAGE_PREFIX=ras-61-unsteady
IMG=$ACCOUNT/$IMAGE_PREFIX:$TAG

# aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $ACCOUNT

# build and push version and latest 
docker build . -t $IMG
docker push $IMG
docker tag $IMG $ACCOUNT/$IMAGE_PREFIX:latest
docker push $ACCOUNT/$IMAGE_PREFIX:latest
