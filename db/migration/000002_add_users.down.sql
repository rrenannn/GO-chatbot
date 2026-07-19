ALTER TABLE customers DROP CONSTRAINT IF EXISTS customers_user_phone_unique;
ALTER TABLE customers ADD CONSTRAINT customers_phone_number_key UNIQUE (phone_number);
ALTER TABLE customers DROP COLUMN IF EXISTS user_id;
DROP TABLE IF EXISTS users;
