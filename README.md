# Rate Limiter Service

This is a simple Rate Limiter Service implemented in Go, supporting Token Bucket and Sliding Window algorithms.

## Functional Requirements

- **Rate Limiting**: The service enforces rate limits on incoming requests based on a key (e.g., IP address or user ID).
- **Algorithms Supported**:
  - **Token Bucket**: Allows a burst of requests up to the capacity, then refills at a constant rate.
  - **Sliding Window**: Tracks requests within a time window and limits the number of requests per window.
- **Request Handling**: Exposes REST APIs for checking rate limits.
- **Response**: JSON responses with appropriate HTTP status codes.

## High-Level Architecture

The Rate Limiter Service follows a layered architecture for modularity, scalability, and maintainability. Below is an overview of the major components and their responsibilities.

### Architecture Diagram (Textual Representation)
```
[Client Applications]
        |
        v
[HTTP Server] <-- Handles incoming requests, routing, and responses
        |
        v
[API Layer] <-- Validates requests, orchestrates rate limit checks, formats responses
        |
        v
[Service Layer] <-- Selects algorithm, applies rate limiting, returns decisions
        |
        v
[Rate Limiter Engine] <-- Core logic for enforcing limits using algorithms
        |
        v
[In-Memory Storage] <-- Stores state (tokens, timestamps) per key
        ^
        |
[Configuration Manager] <-- Loads settings from environment variables
```

### Major Components

1. **Client Applications**:
   - External systems or users making requests to the service.
   - Send HTTP POST requests to the `/api/v1/rate-limit/check` endpoint with a JSON payload containing the key.

2. **HTTP Server**:
   - Built using Go's `net/http` package.
   - Responsibilities:
     - Listens on a configurable port (default 8080).
     - Accepts HTTP requests and routes them to appropriate handlers.
     - Manages response headers (e.g., Content-Type: application/json) and status codes.
     - Handles errors like invalid methods or JSON parsing failures.

3. **API Layer**:
   - Consists of HTTP handlers (e.g., in `main.go`).
   - Responsibilities:
     - Parses incoming JSON requests (e.g., extracts the `key`).
     - Defaults to client IP if no key is provided.
     - Invokes the Service Layer to check allowance.
     - Formats and sends JSON responses with `allowed` and `remaining` fields.

4. **Service Layer**:
   - Encapsulates rate limiting logic (`pkg/service`).
   - Responsibilities:
     - Selects and initializes the appropriate algorithm based on config.
     - Provides a unified `CheckRateLimit` method for decisions.
     - Abstracts algorithm details from the API layer.

5. **Rate Limiter Engine**:
   - Core component implemented via the `RateLimiter` interface and concrete types (`TokenBucket`, `SlidingWindow`).
   - Responsibilities:
     - Enforces rate limits based on selected algorithm.
     - Tracks per-key state (e.g., token counts, request timestamps) via the `Store` interface.
     - Uses the `Clock` interface for time operations.
     - Returns allowance status and remaining capacity.
     - Supports extensibility for new algorithms or storage backends.

5. **In-Memory Storage**:
   - Uses Go maps with TTL-based cleanup.
   - Responsibilities:
     - Stores transient state for rate limiting (no persistence).
     - Automatic cleanup of stale entries to prevent memory leaks.
     - Fast access for low-latency decisions.
     - Limitation: State is lost on restart; suitable for single-instance or stateless deployments.

6. **Configuration Manager**:
   - Embedded in `main.go` via environment variable parsing.
   - Responsibilities:
     - Loads algorithm type, parameters (e.g., capacity, rate), and port from env vars.
     - Initializes the appropriate `RateLimiter` instance at startup.
     - Provides defaults for missing configurations.

### Data Flow
1. Client sends POST request with JSON body to `/api/v1/rate-limit/check`.
2. HTTP Server routes to API Layer handler.
3. API Layer parses request, calls Service Layer with key.
4. Service Layer delegates to Rate Limiter Engine.
5. Engine checks in-memory state, applies algorithm logic, updates state if allowed.
6. Engine returns allowance and remaining; Service Layer returns decision; API Layer formats JSON response.
6. HTTP Server sends response with appropriate status code (200 or 429).

### Key Design Decisions
- **Stateless per Request**: Each check is independent but relies on shared state.
- **In-Memory for Performance**: Trade-off for speed vs. persistence.
- **Interface-Based Design**: Allows swapping algorithms without changing API.
- **Environment Configuration**: Enables container-friendly deployment.
- **Thread Safety**: Mutexes prevent race conditions in concurrent scenarios.

### Interfaces and Packages
The service uses Go interfaces for modularity and testability, organized in separate packages:
- **`pkg/clock`**: `Clock` interface for time operations. `RealClock` implementation.
- **`pkg/store`**: `Store` interface for key-value storage. `InMemoryStore` implementation.
- **`pkg/ratelimiter`**: `RateLimiter` interface for limiting logic. `TokenBucket` and `SlidingWindow` implementations.

This architecture supports the functional requirements while being simple to deploy and extend.

## REST APIs

### Check Rate Limit
- **Endpoint**: `POST /api/v1/rate-limit/check`
- **Description**: Checks if a request is allowed for the given key.
- **Request Body** (JSON):
  ```json
  {
    "key": "string"  // Optional; defaults to client IP if empty
  }
  ```
- **Response**:
  - **200 OK** (Allowed):
    ```json
    {
      "allowed": true,
      "remaining": 5
    }
    ```
  - **429 Too Many Requests** (Denied):
    ```json
    {
      "allowed": false,
      "remaining": 0
    }
    ```
- **Notes**: `remaining` indicates remaining tokens (Token Bucket) or remaining requests (Sliding Window) before limit is hit.

## Non-Functional Requirements

- **Performance**: In-memory storage for low latency.
- **Thread Safety**: Uses mutexes to handle concurrent requests.
- **Configurability**: Configured via environment variables.
- **Scalability**: Single instance; for distributed, use external storage like Redis (not implemented).
- **Reliability**: Simple implementation without persistence; state lost on restart.

## Supported Algorithms

### Token Bucket

- **Parameters**:
  - `CAPACITY`: Maximum number of tokens (default 10).
  - `RATE`: Tokens added per second (default 1).
- **Logic**: Each request consumes a token. Tokens refill over time.

### Sliding Window

- **Parameters**:
  - `WINDOW_SIZE_SECONDS`: Size of the sliding window in seconds (default 60).
  - `MAX_REQUESTS`: Maximum requests per window (default 10).
- **Logic**: Keeps a list of request timestamps per key, removes old ones outside the window.

## Usage

1. Set environment variables:
   - `ALGORITHM`: "tokenbucket" or "slidingwindow" (default "tokenbucket").
   - For Token Bucket: `CAPACITY`, `RATE`.
   - For Sliding Window: `WINDOW_SIZE_SECONDS`, `MAX_REQUESTS`.
   - `PORT`: Server port (default 8080).
   - `TTL_SECONDS`: Time-to-live for in-memory store entries (default 3600 seconds).
   - `MAX_KEYS`: Maximum number of keys in store (default 0, unlimited).

2. Run: `go run ./cmd/ratelimiter/cmd/ratelimiter`

3. Test: `curl -X POST -H "Content-Type: application/json" -d '{"key":"test"}' http://localhost:8080/api/v1/rate-limit/check`

## Running

```bash
export ALGORITHM=tokenbucket
export CAPACITY=5
export RATE=2
go run ./cmd/ratelimiter
```

Then, in another terminal:

```bash
for i in {1..10}; do curl -s -X POST -H "Content-Type: application/json" -d '{"key":"user1"}' http://localhost:8080/api/v1/rate-limit/check | jq .allowed; done
```