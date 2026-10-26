-- Kembalikan ke varchar
ALTER TABLE products ALTER COLUMN status DROP DEFAULT;

ALTER TABLE products
    ALTER COLUMN status TYPE VARCHAR(20)
    USING status::VARCHAR(20);

ALTER TABLE products ALTER COLUMN status SET NOT NULL;

DROP TYPE product_status;