#!/bin/bash

# Needs to be run before building Docker Image
buildkit_version=$(jq -r '.buildkit_version' buildkit/version.json)

docker pull ${buildkit_version}

docker save ${buildkit_version} > buildkit/buildkit.tar
