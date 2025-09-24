# docker-local

Docker Compose configuration to support local testing with AWS services.

## Prerequisites

- Docker

## Usage

1. Start the DynamoDB container in the background: `docker compose -f docker-local/docker-compose.yml up -d`
1. Run the server
   - `AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local AWS_ENDPOINT_URL=http://localhost:8000 AWS_REGION=local-kt go run github.com/signalapp/keytransparency/cmd/kt-server -config ./example/config.yaml`
1. Test with kt-client
1. Stop the server
1. Stop the container
   - `docker compose -f docker-local/docker-compose.yml down`
