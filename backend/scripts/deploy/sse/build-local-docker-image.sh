#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
LOG_FILE=${SCRIPT_DIR}/build-local-docker-image.log
ERROR=""
IMAGE_NAME="sse-gateway-local"

function stop_and_remove_container() {
    # Stop and remove the existing container
    docker stop ${IMAGE_NAME} >/dev/null 2>&1
    docker rm ${IMAGE_NAME} >/dev/null 2>&1
}

function remove_image() {
    # Remove the existing image
    docker rmi ${IMAGE_NAME} >/dev/null 2>&1
}

function build_image() {
    # build docker
    cd ${SCRIPT_DIR}
    docker build ../../../ -f Dockerfile -t ${IMAGE_NAME} || ERROR="build_image failed"
}

function log_message() {
    if [[ ${ERROR} != "" ]];
    then
        >&2 echo "build failed, Please check build-local-docker-image.log for more details"
        >&2 echo "ERROR: ${ERROR}"
        exit 1
    else
        echo "docker image with tag '${IMAGE_NAME}' built successfully."
        echo "This gateway can only reach the backend when sharing its docker network."
        echo "Sample run (attach to the app-tier network where admin-service runs):"
        echo ""
        echo "docker run -d -p 8013:8080 --network app-tier --name ${IMAGE_NAME} ${IMAGE_NAME}"
    fi
}

echo "Info: Stopping and removing existing container and image" | tee ${LOG_FILE}
stop_and_remove_container
remove_image

echo "Info: Building docker image" | tee -a ${LOG_FILE}
build_image 1>> ${LOG_FILE} 2>> ${LOG_FILE}

log_message | tee -a ${LOG_FILE}
