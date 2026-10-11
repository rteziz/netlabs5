# File transfer over TCP

A simple client and server for sending a file over TCP. The server prints the
receive rate for each client.

## Build

Run from the project root (the folder with `../go.mod`):

```
go build -o server ./cmd/server
go build -o client ./cmd/client
```

## Run

Start the server with a port number:

```
./server 9000
```

Send a file with the client: file path, server host (DNS name or IP), port:

```
./client ./photo.jpg localhost 9000
```

The client prints `Transfer successful` or `Transfer FAILED: <reason>`
(exit code 1 on failure).

## What the server does

- Saves files into the `../uploads` folder in its current directory.
  The folder is created automatically.
- If a file with the same name already exists, it saves it as `name_1.ext`,
  `name_2.ext`, and so on. Existing files are never overwritten.
- Only the last part of the file name is used, so nothing can be written
  outside `../uploads`.
- Serves many clients at the same time.
- Every 3 seconds, prints the instant and average rate (bytes per second) for
  each active client. A transfer shorter than 3 seconds is printed once at the end.

Example output:

```
2026/10/06 12:00:03 [127.0.0.1:52144] instant: 91234816 B/s, average: 91234816 B/s
2026/10/06 12:00:05 [127.0.0.1:52144] file received successfully
```

## Limits

- File name: at most 4096 bytes, valid UTF-8.
- File size: at most 1 TiB.
- If anything goes wrong, the server deletes the partial file, tells the client
  the transfer failed, closes that connection and keeps serving the others.

## Protocol

Client to server:

| Field       | Size                | Notes                  |
|-------------|---------------------|------------------------|
| name length | 2 bytes, big-endian | in bytes               |
| file name   | name length bytes   | UTF-8                  |
| file size   | 8 bytes, big-endian | in bytes               |
| contents    | file size bytes     | the file itself        |

Server to client, after the whole file is received:

| Field  | Size   | Notes                      |
|--------|--------|----------------------------|
| status | 1 byte | `0` = success, `1` = failure |

## Project layout

```
go.mod
protocol/protocol.go     wire format shared by client and server
cmd/server/main.go       listens, starts a goroutine per client
cmd/server/handler.go    receives one file and sends the status
cmd/server/storage.go    safe file naming and creation in uploads/
cmd/server/meter.go      speed reporting
cmd/client/main.go       sends the file
```