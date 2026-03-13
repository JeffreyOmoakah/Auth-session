make run            # start the server
make docker-up      # spin up Postgres + Redis
make docker-down    # tear down containers
make migrate-up     # run goose migrations
make migrate-down   # rollback last migration
make migrate-status # check migration state
make sqlc           # regenerate sqlc output from queries