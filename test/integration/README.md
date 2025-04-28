# Receipt API Integration Tests

This directory contains integration tests for the Receipt API based on the OpenAPI specification. These tests verify that all API endpoints work as expected.

## Test Coverage

The integration tests cover the following endpoints:

- `POST /receipts/scan` - Scan a receipt image to extract transaction data
- `POST /receipts` - Create a receipt manually
- `GET /receipts` - List all receipts with pagination and filtering
- `GET /receipts/{receiptId}` - Get a receipt by ID
- `PUT /receipts/{receiptId}` - Update a receipt
- `DELETE /receipts/{receiptId}` - Delete a receipt
- `GET /receipts/{receiptId}/items` - Get receipt items
- `GET /dashboard/summary` - Get dashboard summary
- `GET /dashboard/spending-trends` - Get spending trends
- `GET /insights/spending-by-category` - Get spending by category
- `GET /insights/merchant-frequency` - Get merchant frequency
- `GET /insights/monthly-comparison` - Get monthly comparison

## Prerequisites

1. The Receipt API server must be running on `http://localhost:8080` or the URL specified by the `API_BASE_URL` environment variable.
2. A sample receipt image for scanning should be placed in the `testdata` directory as `sample_receipt.jpg`.

## Running the Tests

To run the integration tests:

```bash
# Make sure the server is running first
cd /path/to/invoice-processor-service
go run cmd/server/main.go

# In another terminal, run the tests
cd /path/to/invoice-processor-service
go test -v ./test/integration/...
```

You can also specify a different API base URL:

```bash
API_BASE_URL=http://localhost:8081/v1 go test -v ./test/integration/...
```

## Test Flow

1. Create a test receipt
2. Try to scan a receipt image (if available)
3. List all receipts
4. Get the test receipt by ID
5. Get items for the test receipt
6. Update the test receipt
7. Test all dashboard and insights endpoints
8. Delete the test receipt
9. Verify deletion

## Notes

- These tests do not use mocks - they perform real API calls to verify functionality.
- The tests run in a specific order to ensure dependencies between tests are satisfied.
- Test data is cleaned up at the end of the test suite.

## Troubleshooting

If you encounter errors like "connection refused", make sure the API server is running.

If the scan receipt test fails with a 422 status code, this is expected if the sample image cannot be processed. The test will continue as this is a valid response from the API.
