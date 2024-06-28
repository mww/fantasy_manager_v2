To create a postgres DB in docker

Note: this changes the default port to avoid conflicts with other projects.

```
docker run --name fantasy-manager-db \
    -p 5433:5432 \
    -v ./schema/:/docker-entrypoint-initdb.d/ \
    -e POSTGRES_USER=ffuser \
    -e POSTGRES_PASSWORD=secret \
    -e POSTGRES_DB=fantasy_manager \
    -d postgres:16.3-alpine
```

To connect to psql for the running database server

```
docker exec -it fantasy-manager-db psql -U ffuser fantasy_manager
```
