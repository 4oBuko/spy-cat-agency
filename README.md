# Spy-cat-agency

A REST API to manage your cats and their missions

## Requirements

- Golang 1.24
- Docker Compose

## How to Run

### How to Run Tests

To run tests for this app you need to execute command bellow in the ROOT of the project. There is a mysql testcontainer for tests and it may take some time to pull a docker image and to start a container. Containers starts in 8-11 seconds.

```bash
go test ./...
```

### Run the App

Run all commands in the ROOT of the project. App doesn't require any env variable to simplify process of running the app. db connection url and breeds api are in the main.go file.

Start test database with the following command:

```bash
docker-compose up -d
```

Then execute the following command:

```bash
go run ./cmd/api/main.go
```
### Postman group
To run request using postman you import [postman-collection.json](spy-cat-agency.postman_collection.json) in your postman client. Execute requests in folders one by one.
