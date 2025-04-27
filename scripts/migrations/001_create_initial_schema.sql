-- Create receipts table
CREATE TABLE IF NOT EXISTS receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant VARCHAR(255) NOT NULL,
    date DATE NOT NULL,
    total DECIMAL(10, 2) NOT NULL,
    tax DECIMAL(10, 2),
    subtotal DECIMAL(10, 2),
    image_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create receipt_items table with foreign key to receipts
CREATE TABLE IF NOT EXISTS receipt_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_id UUID NOT NULL REFERENCES receipts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    qty INTEGER NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    category VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on receipt_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_receipt_items_receipt_id ON receipt_items(receipt_id);

-- Create index on merchant for faster filtering
CREATE INDEX IF NOT EXISTS idx_receipts_merchant ON receipts(merchant);

-- Create index on date for faster filtering and sorting
CREATE INDEX IF NOT EXISTS idx_receipts_date ON receipts(date);

-- Create index on category for insights queries
CREATE INDEX IF NOT EXISTS idx_receipt_items_category ON receipt_items(category);

-- Add trigger for updated_at timestamp on receipts
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_receipts_modtime
BEFORE UPDATE ON receipts
FOR EACH ROW
EXECUTE FUNCTION update_modified_column();

-- Add trigger for updated_at timestamp on receipt_items
CREATE TRIGGER update_receipt_items_modtime
BEFORE UPDATE ON receipt_items
FOR EACH ROW
EXECUTE FUNCTION update_modified_column();
