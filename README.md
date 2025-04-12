# auth-rest-api

- This is a simple REST API using golang and redis for JWT token based authentication.
- The API has endpoints for user registration, login, and protected routes for token refresh and revoke.

## CI/CD Pipeline

This project uses GitHub Actions for continuous integration and deployment:

1. **Build and Test**: On every push and pull request, the pipeline:
   - Sets up Go environment
   - Installs dependencies
   - Runs linters (golangci-lint)
   - Runs tests with race detection and coverage reporting
   - Uploads coverage to Codecov

2. **Docker Build**: On pushes to the main branch, the pipeline:
   - Builds a Docker image
   - Pushes the image to Docker Hub

To use the CI/CD pipeline:
1. Set up the following secrets in your GitHub repository:
   - `DOCKER_USERNAME`: Your Docker Hub username
   - `DOCKER_PASSWORD`: Your Docker Hub password/token

2. The pipeline will automatically run on:
   - Pull requests to the main branch
   - Pushes to the main branch

## How to use this api

### Method 1

1. Clone the repository
2. install golang preferable v1.24
3. set envs in .env file  according to your or use existing .env
4. run the redis-server on your local machine at `localhost:6379`
5. run the application by command `go run cmd/main.go`
6. you can see logs and run the curl commands in the terminal or from postman !!

### Method 2

1. Clone the repository
2. make sure you have `docker` & `docker-compose` installed locally
3. make sure port `9001` and `6379` is free to run api and redis
4. run the command `docker-compose up` and wait till it completes
5. you can try the app at `localhost:9001` if you don't change any env

## Development

This project includes a Makefile for common development tasks:

```bash
# Build the application
make build

# Run tests
make test

# Run linter
make lint

# Run the application
make run

# Build Docker image
make docker-build

# Run Docker container
make docker-run

# Development with hot reload
make dev
```

## Exposed Endpoints

1. **POST /signup**: Register a new user with email & password
2. **POST /signin**: Login with registered user need email & password
3. **POST /refresh**: Refresh token before expiry of access token, *needs access-token in authentication header & refreshToken as json-body*
4. **POST /revoke**: Revoke the access token, *needs access-token in authentication header*

NOTE: **password** should be 8 character long, **email** should be in format `user@example.com` must have`@` and `.` in it

## Curls for testing

- See [API specification](./openapi/auth-rest-api.yaml)

- Register a new user (change email & password value as needed): `curl --location 'http://localhost:9001/signup' --header 'Content-Type: application/json' --data-raw '{ "email":"sumit@kumar.com", "password":"sumit@kumar" }'`

- Login to get JWT token: `curl --location 'http://localhost:9001/signin' --header 'Content-Type: application/json' --data-raw '{"email":"sumit@kumar.com", "password":"sumit@kumar"}'`

- Refresh existing token (before auth token expiry): `curl --location 'http://localhost:9001/refresh' --header 'Content-Type: application/json' --header 'Authorization: ******' --data '{
   "refreshToken": "<put refresh token here>"
}'`
  - **NOTE**: *replace `********` and `<put refresh token here>` with actual values from previous signin*
  - *Access token in Authorization Header and refresh token in request body*

- Revoke existing token: `curl --location --request POST 'http://localhost:9001/revoke' --header 'Authorization: *****'`
  - **NOTE**: *replace `*****` with actual access token value (it is a bearer token)*
