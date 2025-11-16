# Swagger Auto-Regeneration in Development

## Overview
This guide explains how to automatically regenerate Swagger documentation when you make changes to API annotations, request/response structs, or handler code.

## Using wgo with Auto-Regeneration

The `make hotreload` command automatically regenerates Swagger docs before starting the server.

### Usage

1. **Start development server with auto-reload and auto-swagger:**
   ```bash
   make hotreload
   ```

2. **What happens:**
   - Watches all `.go` files for changes
   - When a change is detected:
     - Regenerates Swagger docs (`~/go/bin/swag init`)
     - Restarts the server
   - Access Swagger UI at: `http://localhost:8080/swagger/index.html`

3. **Make changes and see them reflected:**
   - Edit API annotations in handlers
   - Modify request/response structs in `internal/domain/`
   - Update descriptions, parameters, etc.
   - Save the file
   - Swagger docs regenerate automatically
   - Refresh browser to see changes

### Example Workflow

```bash
# Start dev server with auto-reload
make hotreload

# Make changes to your code
# Edit internal/handler/receipt_handler.go
# Change: @Summary Scan a receipt image
# To:     @Summary Scan and extract receipt data

# Save the file
# wgo detects change → regenerates Swagger → restarts server
# Refresh http://localhost:8080/swagger/index.html to see changes
```

## What Triggers Swagger Regeneration?

Changes to these files will trigger regeneration:

1. **Handler files** (`internal/handler/*.go`)
   - API endpoint annotations
   - Route definitions
   - Request/response handling

2. **Domain models** (`internal/domain/*.go`)
   - Struct definitions used in requests/responses
   - Field tags and validation

3. **Main file** (`cmd/server/main.go`)
   - General API information
   - Host, schemes, security definitions

## Common Development Scenarios

### Scenario 1: Changing API Description
```go
// Before
// @Summary Get receipts
// @Description Retrieve all receipts

// After
// @Summary List all receipts with filters
// @Description Get a paginated list of receipts with optional date and merchant filters
```
**Result:** Save → Auto-regenerate → Refresh browser

### Scenario 2: Adding New Query Parameter
```go
// Add new parameter annotation
// @Param category query string false "Category filter"
```
**Result:** Save → Auto-regenerate → New parameter appears in Swagger UI

### Scenario 3: Modifying Response Struct
```go
// internal/domain/receipt.go
type Receipt struct {
    ID       string    `json:"id"`
    Merchant string    `json:"merchant"`
    NewField string    `json:"newField"` // Added this
    // ...
}
```
**Result:** Save → Auto-regenerate → Schema updated in Swagger UI

### Scenario 4: Adding New Endpoint
```go
// @Summary New endpoint
// @Description Does something new
// @Tags receipts
// @Router /v1/receipts/new-endpoint [post]
func (h *ReceiptHandler) NewEndpoint(c *gin.Context) {
    // implementation
}
```
**Result:** Save → Auto-regenerate → New endpoint appears in Swagger UI

## Troubleshooting

### Swagger docs not updating?

1. **Check if swag is installed:**
   ```bash
   ~/go/bin/swag --version
   ```

2. **Manually regenerate to see errors:**
   ```bash
   make swagger
   ```

3. **Check for annotation syntax errors:**
   - Missing `@` prefix
   - Incorrect parameter format
   - Invalid route syntax

### Browser showing old docs?

1. **Hard refresh:** `Cmd+Shift+R` (Mac) or `Ctrl+Shift+R` (Windows/Linux)
2. **Clear browser cache**
3. **Check if docs were actually regenerated:**
   ```bash
   ls -la docs/
   # Check modification time of docs.go
   ```

### wgo not detecting changes?

1. **Verify wgo is installed:** `~/go/bin/wgo --version`
2. **Check file extensions:** wgo watches `.go` files by default
3. **Restart the hotreload:** Stop and run `make hotreload` again

## Performance Tips

1. **Ignore generated files** to prevent infinite loops:
   - The Makefile command excludes test files and common directories

2. **wgo has built-in debouncing** to prevent too-frequent regeneration

## Best Practices

1. **Always annotate new endpoints** before testing
2. **Keep annotations up-to-date** with implementation
3. **Test in Swagger UI** after making changes
4. **Commit generated docs** to version control
5. **Document complex request/response structures** with examples

## Quick Reference

| Command | Purpose |
|---------|---------|
| `make hotreload` | Start dev server with auto-reload + auto-swagger |
| `make swagger` | Manually regenerate Swagger once |
| `~/go/bin/swag fmt` | Format swagger annotations |

## Additional Resources

- [Swaggo Documentation](https://github.com/swaggo/swag)
- [Swagger Annotation Examples](https://github.com/swaggo/swag#declarative-comments-format)
- [OpenAPI Specification](https://swagger.io/specification/)
