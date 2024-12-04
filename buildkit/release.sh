#!/bin/bash

# Path to the JSON file
json_file="buildkit/version.json"

# Extract the image name using jq (ensure jq is installed)
image_name=$(jq -r '.buildkit_version' "$json_file")

# Check if image name was extracted successfully
if [ -z "$image_name" ]; then
  echo "Error: Unable to extract image name from JSON file."
  exit 1
fi

# Check for an optional platform override
platform_override=${1:-}
if [ -n "$platform_override" ]; then
  echo "Using platform override: $platform_override"
else
  echo "No platform override provided. Using default platform."
fi

# Pull the Docker image with optional platform specification
echo "Pulling Docker image: $image_name"
if [ -n "$platform_override" ]; then
  docker pull --platform "$platform_override" "$image_name"
else
  docker pull "$image_name"
fi

# Save the Docker image to a tarball
tar_file="buildkit/buildkit.tar"
echo "Saving Docker image to tarball: $tar_file"
docker save "$image_name" -o "$tar_file"

echo "Done. Docker image saved to $tar_file"
