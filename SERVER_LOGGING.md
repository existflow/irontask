# Server Logging Enhancement

## Changes Made

Added comprehensive HTTP request/response logging to the IronTask sync server with **console-only output** (no file logging).

### Files Modified

1. **[server/server.go](file:///Users/tphuc/coding/iron/irontask/server/server.go)**
   - Added logger import
   - Replaced Echo's default request logger with custom middleware
   - Logs both request and response with full details

2. **[cmd/irontask-server/main.go](file:///Users/tphuc/coding/iron/irontask/cmd/irontask-server/main.go)**
   - Added logger initialization (console-only)
   - Environment variable support for `LOG_LEVEL`
   - Server lifecycle logging

## Logging Output Format

### Console Output
```
REQUEST: POST /api/v1/magic-link  status=200  size=45  duration=12.5ms
REQUEST: GET /api/v1/magic-link/932486fab7d9227aaf60016949176f336c0511f158cfa76ae41e1d20d712b785  status=200  size=128  duration=5.2ms
REQUEST: GET /api/v1/sync?since=0  status=200  size=1024  duration=25.8ms
REQUEST: GET /api/v1/sync?since=37  status=200  size=512  duration=15.3ms
```

### Structured Log Output (stderr)
```
[2026-01-11 16:55:00.123] INFO server.go:60: HTTP Request | method=POST uri=/api/v1/magic-link remote=127.0.0.1:54321
[2026-01-11 16:55:00.135] INFO server.go:70: HTTP Response | method=POST uri=/api/v1/magic-link status=200 size=45 duration=12.5ms
[2026-01-11 16:55:01.456] INFO server.go:60: HTTP Request | method=GET uri=/api/v1/magic-link/932486fab7d9227aaf60016949176f336c0511f158cfa76ae41e1d20d712b785 remote=127.0.0.1:54322
[2026-01-11 16:55:01.461] INFO server.go:70: HTTP Response | method=GET uri=/api/v1/magic-link/932486fab7d9227aaf60016949176f336c0511f158cfa76ae41e1d20d712b785 status=200 size=128 duration=5.2ms
[2026-01-11 16:55:02.789] INFO server.go:60: HTTP Request | method=GET uri=/api/v1/sync?since=0 remote=127.0.0.1:54323
[2026-01-11 16:55:02.815] INFO server.go:70: HTTP Response | method=GET uri=/api/v1/sync?since=0 status=200 size=1024 duration=25.8ms
[2026-01-11 16:55:03.123] INFO server.go:60: HTTP Request | method=GET uri=/api/v1/sync?since=37 remote=127.0.0.1:54324
[2026-01-11 16:55:03.138] INFO server.go:70: HTTP Response | method=GET uri=/api/v1/sync?since=37 status=200 size=512 duration=15.3ms
```

## Features

✅ **HTTP Method** - Shows GET, POST, etc.
✅ **Request URI** - Full URI including query parameters
✅ **Status Code** - HTTP response status
✅ **Response Size** - Size in bytes
✅ **Duration** - Request processing time
✅ **Remote Address** - Client IP and port
✅ **Console Output** - All logs to stdout/stderr (no file)
✅ **Environment Variable** - `LOG_LEVEL` for controlling verbosity

## Usage

### Running the Server

```bash
# Default (INFO level)
./irontask-server

# Debug mode
LOG_LEVEL=DEBUG ./irontask-server

# Custom port
PORT=3000 ./irontask-server
```

### Output Redirection (Optional)

If you want to save logs to a file, you can redirect output:

```bash
# Save all output to file
./irontask-server > server.log 2>&1

# Save only errors to file
./irontask-server 2> errors.log
```

## Benefits

1. **Complete Request Tracking** - See method, URI, and all parameters
2. **Performance Monitoring** - Duration shows how long each request takes
3. **Response Validation** - Status codes and sizes help debug issues
4. **Client Information** - Remote address helps track request sources
5. **Real-time Visibility** - Console output for immediate feedback
6. **No File Management** - No need to rotate or manage log files
