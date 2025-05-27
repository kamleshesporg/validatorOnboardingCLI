#!/bin/bash


echo "Mrmintchain Binary files downloading..."

#mrmintchain binary fetching...
curl -sSLO https://raw.githubusercontent.com/kamleshesporg/validatorOnboardingCLI/main/mrmintchain
#mrmintd binary fetching...
curl -sSLO https://raw.githubusercontent.com/kamleshesporg/validatorOnboardingCLI/main/chain/ethermintd

CONTAINER_CLI=docker

image_tag=latest
 

echo "===> Pulling mrmintchain Image"
local image_name="docker.io/kamleshesp/mrmintchain:${image_tag}"

echo "====>  ${image_name}"
${CONTAINER_CLI} pull "${image_name}"


