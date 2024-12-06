# Fantasy Manager

This is a program I have written mostly for myself to help me generate custom power rankings for my various fantasy football leagues. Why? Because power rankings are fun. They can be completely wrong, but if they give people in the league something to complain about, or something to rally around and generate more conversations then they have done their job.

If this is useful to anyone else that is great, but it is built around my own needs.

# Development

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

Be sure to look at `.env.sample`. You need to set POSTGRES_CONN_STR either
as an environment variable or in your `.env` file before you can run the
server.

# Code Coverage

To generate code coverage run

```
go test -count=1 -race -shuffle=on -coverprofile=./cover.out -covermode=atomic -coverpkg=./... ./...
```

Then to view the coverage report

```
go tool cover -html="cover.out"
```
