-- Add currency column to receipt_items table (nullable first)
ALTER TABLE receipt_items 
ADD COLUMN IF NOT EXISTS currency VARCHAR(10);

-- Update existing rows with default currency if NULL
UPDATE receipt_items 
SET currency = 'IDR' 
WHERE currency IS NULL;

-- Make the column non-nullable
ALTER TABLE receipt_items 
ALTER COLUMN currency SET NOT NULL;

-- Set default value for future inserts
ALTER TABLE receipt_items 
ALTER COLUMN currency SET DEFAULT 'IDR';

-- Add comment to explain the column
COMMENT ON COLUMN receipt_items.currency IS 'Currency code (e.g., IDR, USD, EUR)';
