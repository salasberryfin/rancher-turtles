#!/bin/bash

# Run your command and capture its output
output=$(make docker-list-all REGISTRY="$1" ORG="$2" TAG="$3")
PASSPHRASE="$4"

# Use a for loop to iterate over each line
IFS=$'\n'       # Set the Internal Field Separator to newline
line_count=0    # Counter to keep track of the current line
total_lines=$(echo "$output" | wc -l)  # Get the total number of lines
githubimageoutput=("multiarch_image" "amd64_image" "arm64_image" "s390x_image")
githubdigestoutput=("multiarch_digest" "amd64_digest" "arm64_digest" "s390x_digest")

for line in $output; do
  # Run the Docker command and get the digest
  digest=$(docker buildx imagetools inspect "$line" --format '{{json .}}' | jq -r .manifest.digest)

  # Add encrypted image name to the output
  image_output="$line"
  #encrypted_image=$(gpg --symmetric --batch --passphrase ${PASSPHRASE} --output - <(echo ${image_output}) | base64 -w0)
  #echo "${githubimageoutput[$line_count]}=${encrypted_image}" >> "$GITHUB_OUTPUT"
  echo "::add-mask::${image_output}"
  echo "${githubimageoutput[$line_count]}=${image_output}"
  # Add encrypted digest to the output
  digest_output="$digest"
  #encrypted_digest=$(gpg --symmetric --batch --passphrase ${PASSPHRASE} --output - <(echo ${digest_output}) | base64 -w0)
  #echo "${githubdigestoutput[$line_count]}=${encrypted_digest}" >> "$GITHUB_OUTPUT"
  echo "::add-mask::${digest_output}"
  echo "${githubdigestoutput[$line_count]}=${digest_output}"

  # Increment the line counter
  line_count=$((line_count + 1))
done
