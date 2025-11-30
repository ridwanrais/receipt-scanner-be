-- Add receipt_url column to receipts table
-- This will store the URL of the original receipt image
ALTER TABLE receipts 
ADD COLUMN IF NOT EXISTS receipt_url TEXT;

-- Add comment to explain the column
COMMENT ON COLUMN receipts.receipt_url IS 'URL of the original receipt image for retry scanning';
