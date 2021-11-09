# syntax=docker/dockerfile:1

# This package contains most of the required python libraries.
FROM python:3.8-slim-buster

WORKDIR /app

# Install all requirements
COPY deduplicator/requirements.txt requirements.txt
RUN pip install -r requirements.txt

# Copy and install 
COPY . .

# Run application
CMD [ "python3", "deduplicator.py" ]
