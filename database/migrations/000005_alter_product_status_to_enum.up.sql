-- Buat enum type dulu
CREATE TYPE product_status AS ENUM ('PENDING', 'APPROVED', 'REVISION', 'REJECTED');

-- Hapus default lama dan CHECK constraint (kalau ada) sebelum ubah tipe
ALTER TABLE products ALTER COLUMN status DROP DEFAULT;

-- Ubah tipe kolom dari varchar ke enum
ALTER TABLE products
    ALTER COLUMN status TYPE product_status
    USING status::product_status;

-- Set ulang default dengan tipe enum
ALTER TABLE products ALTER COLUMN status SET NOT NULL;