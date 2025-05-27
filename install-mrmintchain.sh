#!/bin/bash


echo "Mrmintchain Binary files downloading..."

echo "1. mrmintchain binary fetching..."
curl -sSLO https://raw.githubusercontent.com/kamleshesporg/validatorOnboardingCLI/main/mrmintchain

echo "2. mrmintd binary fetching..."
curl -sSLO https://raw.githubusercontent.com/kamleshesporg/validatorOnboardingCLI/main/chain/ethermintd

echo -e "\xE2\x9C\x94 Binary files downloaded!"


CONTAINER_CLI=docker
image_tag=latest
 

echo "===> Pulling mrmintchain Image"
image_name="docker.io/kamleshesp/mrmintchain:${image_tag}"

echo "====>  ${image_name}"
${CONTAINER_CLI} pull "${image_name}"

echo -e "\xE2\x9C\x94 Congratulations mrmintchain installed!"
