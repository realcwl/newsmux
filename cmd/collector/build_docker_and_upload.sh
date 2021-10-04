#!/bin/bash

# Obtain credentials
aws ecr get-login-password --region us-west-1 | docker login --username AWS --password-stdin 213288384225.dkr.ecr.us-west-1.amazonaws.com/data_collector_simple_2

# Build with build context as project root
docker build -t data_collector_simple_2 -f Dockerfile ../../

docker tag data_collector_simple_2:latest 213288384225.dkr.ecr.us-west-1.amazonaws.com/data_collector_simple_2:latest

docker push 213288384225.dkr.ecr.us-west-1.amazonaws.com/data_collector_simple_2:latest