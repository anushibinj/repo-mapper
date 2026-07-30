CREATE TABLE customers (
    id BIGINT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL
);

CREATE TABLE invoices (
    id BIGINT PRIMARY KEY,
    customer_id BIGINT,
    customer_name VARCHAR(255),
    amount DECIMAL(10, 2),
    FOREIGN KEY (customer_id) REFERENCES customers(id)
);
