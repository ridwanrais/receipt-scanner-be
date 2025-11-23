# API Request/Response Logging

This application includes comprehensive request/response logging middleware that captures all API calls with automatic credential redaction.

## Features

- **Complete Request/Response Capture**: Logs all HTTP requests and responses including headers, body, query parameters, and metadata
- **Automatic Credential Redaction**: Sensitive fields are automatically replaced with `[REDACTED]`
- **Dual Format Support**: Choose between JSON (machine-readable) or Pretty (human-readable) formats
- **Performance Metrics**: Includes latency tracking for each request
- **Error Tracking**: Captures and logs any errors that occur during request processing

## Configuration

Configure logging behavior using environment variables:

```bash
# Log format: "json" or "pretty" (default: "json")
LOG_FORMAT=json

# Log level: "debug", "info", "warn", "error" (default: "info")
LOG_LEVEL=info
```

## Log Formats

### JSON Format (Default)

Machine-readable format suitable for log aggregation systems:

```json
{
  "timestamp": "2024-11-23T12:13:45Z",
  "method": "POST",
  "path": "/v1/receipts/scan",
  "status_code": 200,
  "latency": "123.456ms",
  "client_ip": "192.168.1.1",
  "user_agent": "Mozilla/5.0...",
  "headers": {
    "Content-Type": "application/json",
    "Authorization": "[REDACTED]"
  },
  "request_body": {
    "image_url": "https://example.com/receipt.jpg",
    "api_key": "[REDACTED]"
  },
  "response_body": {
    "receipt_id": "12345",
    "status": "processed"
  }
}
```

### Pretty Format

Human-readable format with visual separators and emojis:

```
================================================================================
🕐 Timestamp: 2024-11-23T12:13:45Z
📍 POST /v1/receipts/scan
📊 Status: 200 | ⏱️  Latency: 123.456ms
🌐 Client IP: 192.168.1.1

📋 Headers:
  Content-Type: application/json
  Authorization: [REDACTED]

📤 Request Body:
  {
    "image_url": "https://example.com/receipt.jpg",
    "api_key": "[REDACTED]"
  }

📥 Response Body:
  {
    "receipt_id": "12345",
    "status": "processed"
  }
================================================================================
```

## Sensitive Field Redaction

The following fields and headers are automatically redacted:

### Headers
- Authorization
- API-Key / Api-Key / ApiKey
- Token
- Secret
- Password
- Bearer
- Cookie
- Session

### Body Fields
- password
- token
- api_key / apikey / api-key
- secret
- authorization
- auth
- bearer
- key
- credential
- access_token
- refresh_token
- session
- cookie

The redaction is case-insensitive and works with nested JSON structures.

## Log Entry Structure

Each log entry includes:

- **timestamp**: ISO 8601 formatted timestamp
- **method**: HTTP method (GET, POST, PUT, DELETE, etc.)
- **path**: Request path
- **status_code**: HTTP response status code
- **latency**: Request processing time
- **client_ip**: Client IP address
- **user_agent**: Client user agent string
- **request_id**: Optional request ID for tracing
- **headers**: Request headers (sensitive ones redacted)
- **query_params**: URL query parameters
- **request_body**: Request body (sensitive fields redacted)
- **response_body**: Response body (sensitive fields redacted)
- **error**: Any errors that occurred during processing

## Usage Examples

### Development (Pretty Format)

```bash
export LOG_FORMAT=pretty
export LOG_LEVEL=debug
go run cmd/server/main.go
```

### Production (JSON Format)

```bash
export LOG_FORMAT=json
export LOG_LEVEL=info
go run cmd/server/main.go
```

### With Docker

```dockerfile
ENV LOG_FORMAT=json
ENV LOG_LEVEL=info
```

## Best Practices

1. **Production**: Use JSON format for easier parsing and integration with log management systems
2. **Development**: Use pretty format for easier debugging and readability
3. **Log Level**: Use "info" or "warn" in production to reduce log volume
4. **Sensitive Data**: The middleware automatically redacts common sensitive fields, but always review your logs to ensure no sensitive data is exposed

## Integration with Log Management Systems

The JSON format is compatible with popular log management systems:

- **ELK Stack**: Direct ingestion via Filebeat
- **Splunk**: JSON parsing enabled by default
- **CloudWatch**: Use AWS CloudWatch agent
- **Datadog**: Use Datadog agent with JSON log parsing
- **Grafana Loki**: Use Promtail with JSON pipeline

## Performance Considerations

The logging middleware:
- Captures request/response bodies in memory
- Adds minimal latency (typically < 1ms)
- For large payloads (> 1MB), consider implementing size limits
- Bodies are parsed once and cached for the duration of the request

## Troubleshooting

### Logs not appearing
- Check `LOG_FORMAT` and `LOG_LEVEL` environment variables
- Ensure middleware is registered in server setup
- Verify stdout is not being redirected

### Sensitive data still visible
- Add custom patterns to `sensitiveFields` or `sensitiveHeaderPatterns` in `internal/middleware/logger.go`
- Report any missed patterns as security issues

### Performance issues
- Consider implementing body size limits for large payloads
- Use "warn" or "error" log level to reduce volume
- Disable request/response body logging for specific endpoints if needed
