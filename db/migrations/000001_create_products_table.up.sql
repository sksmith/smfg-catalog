CREATE TABLE products
(
    sku  VARCHAR(50) PRIMARY KEY,
    upc  VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100)       NOT NULL
);

COMMIT;
