-- Add sprite caching columns to pokemon table
ALTER TABLE pokemon ADD COLUMN sprite_data BLOB;
ALTER TABLE pokemon ADD COLUMN sprite_content_type TEXT;
ALTER TABLE pokemon ADD COLUMN sprite_fetched_at DATETIME;

-- Create index for sprite fetching
CREATE INDEX IF NOT EXISTS idx_pokemon_sprite_fetched ON pokemon(sprite_fetched_at);