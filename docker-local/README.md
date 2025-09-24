# docker-local

Docker Compose configuration to support local testing with AWS services.

## Prerequisites

- Docker

## Usage

1. Start the DynamoDB container in the background:

   ```shell
   docker compose -f docker-local/docker-compose.yml up -d
   ```

1. Test with kt-client:

   ```shell
   docker compose -f docker-local/docker-compose.yml run -it client bash
   kt-client <...>
   ```

1. Stop the server
1. Stop the container
   - `docker compose -f docker-local/docker-compose.yml down`
