-- Add multi_signatures column to power_of_attorney table
ALTER TABLE power_of_attorney
ADD COLUMN IF NOT EXISTS multi_signatures JSONB;

-- Add comment explaining the column
COMMENT ON COLUMN power_of_attorney.multi_signatures IS 'Stores partial signatures for multi-signature verification';
