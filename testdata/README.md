# Test Data Directory

This directory contains sample data for running integration tests.

## Sample Receipt Image

To test the receipt scanning functionality, we use the sample receipt image file named `sample_receipt.png` in this directory.

You can use any receipt image for testing. If the API is unable to process the image, the test will still pass but will skip the scanning verification.

## Testing without a Sample Image

If no sample receipt image is available, the scan receipt test will be skipped, but all other endpoint tests will still run.
