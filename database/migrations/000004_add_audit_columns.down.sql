ALTER TABLE categories
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS updated_by;

ALTER TABLE products
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS updated_by;