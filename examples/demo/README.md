# GAuth Docker Compose Demo

This demo shows how to run the GAuth service with a Redis-backed token store, along with an example client, using Docker Compose.

## Prerequisites
- Docker and Docker Compose installed

## Services
- **redis**: In-memory data store for token storage
- **gauth-demo**: GAuth demo server (must be built with Redis token store support)
- **example-client**: Example client or web integration (optional, replace with your own)

## Usage

1. **Build the demo server and client images**
   
   Ensure you have a Dockerfile in the demo server root and in the `client/` directory.

   ```sh
   docker-compose build
   ```

2. **Start the services**

   ```sh
   docker-compose up
   ```

   This will launch Redis, the GAuth demo server, and the example client.

3. **Access the services**
   - GAuth demo API: http://localhost:8080
   - Redis: localhost:6379
   - Example client: (see logs or client instructions)

4. **Stop the services**

   ```sh
   docker-compose down
   ```

## Customization
- Edit the `docker-compose.yml` to change ports, environment variables, or add more services.
- Replace the `example-client` with your own integration or web UI.

## Notes
- The demo server must be configured to use Redis via the `REDIS_URL` environment variable.
- For SQL storage, replace the Redis service with a SQL database and update the demo server config accordingly.

---

For more details, see the main project README or the demo server documentation.
