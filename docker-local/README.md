# docker-local

Docker Compose configuration to support local testing with AWS services.

## Prerequisites

- Docker

## Usage

1. Start the DynamoDB container in the background:

   ```shell
   docker compose -f docker-local/docker-compose.yml up -d
   ```

1. Test with kt-client. Run the client:

   ```shell
   docker compose -f docker-local/docker-compose.yml run -it --rm client
   ```

   In the created shell, run commands on the client:

   ```shell
   kt-client -config /src/config.yaml -query-addr localhost:8000 -test-addr localhost:<...> <...>
   ```

1. Stop the containers
   - Kill the shell with `exit`
   - Stop the server with `docker compose -f docker-local/docker-compose.yml down`
