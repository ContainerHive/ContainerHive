#!/bin/bash

# Get the host IP address
HOST_IP=$(ipconfig getifaddr en0)

# Check if HOST_IP is empty
echo "Host IP: $HOST_IP"

# Export the HOST_IP as an environment variable
export HOST_IP

# Start the Docker Compose services
docker-compose up