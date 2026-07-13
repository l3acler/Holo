# Holo

> **Project a fully working REST API from a single JSON file in seconds.**

Holo is a zero-config, single-binary mock server CLI tool written in Go. It acts as a zero-dependency successor to tools like `json-server`, allowing frontend developers and automated UI tests to easily interact with a dynamic, thread-safe mock REST API without running any heavy databases or backend frameworks.

## Features

- **Zero Dependencies:** Written purely in Go using the standard library. Fast, lightweight, and completely standalone.
- **Dynamic Wildcard Routing:** Fully dynamic `/{resource}` and `/{resource}/{id}` endpoints powered by Go 1.25+.
- **Atomic File Persistence:** Mutations are safely persisted back to your JSON file instantly, without data corruption.
- **Thread-Safe:** Safe to use with concurrent testing frameworks.
- **CORS Support:** Integrated out-of-the-box for seamless frontend browser requests.

## Installation

Holo is distributed as a standalone CLI executable. **You do not need to clone the repository to run it.**

**For Go Developers:**
You can install it globally via the Go toolchain:
```bash
go install github.com/l3acler/holo@latest
```

**For Frontend Developers:**
Download the pre-compiled binary for your OS (macOS, Windows, Linux) from the [Releases](https://github.com/l3acler/holo/releases) tab, place it in your project directory, and run it as `./holo`.

## Quick Start

**Step 1:** Create a JSON database file (or simply let Holo create an empty one for you).
```json
{
  "users": [
    { "id": "1", "name": "Bader" }
  ]
}
```

**Step 2:** Start the server.
```bash
holo
```
*(Note: If you downloaded the binary locally, run `./holo`)*

**Step 3:** Fetch or mutate the data using standard REST requests!
```bash
curl http://localhost:8080/users/1
```

## CLI Flags

Holo can be configured quickly at startup via flags:

- `--port` (default: `8080`): The port to run the server on. (e.g. `holo --port 3000`)
- `--file` (default: `db.json`): The path to your JSON database. If the file does not exist, Holo will auto-create it with an empty `{}` object! (e.g. `holo --file store.json`)
- `--memory-only` (default: `false`): If set to `true`, mutations will only happen in memory and will NOT persist back to the JSON file. (e.g. `holo --memory-only=true`)

### API Endpoints

Given the `users` resource in the example above, Holo exposes the following dynamically:

- **`GET /users`:** List all users.
- **`POST /users`:** Add a new user (auto-generates an ID if missing).
- **`GET /users/{id}`:** Get a single user.
- **`PUT /users/{id}`:** Completely replace a user (keeps the original ID).
- **`PATCH /users/{id}`:** Partially update a user's fields.
- **`DELETE /users/{id}`:** Delete a user.