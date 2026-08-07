# Database

## customers

Mapped entity: `Customer`

| Column | Type | PK | FK |
|---|---|---|---|
| email | VARCHAR(255) |  |  |
| id | BIGINT | ✓ |  |
| name | VARCHAR(255) |  |  |

## invoices

Mapped entity: `Invoice`

| Column | Type | PK | FK |
|---|---|---|---|
| amount | DECIMAL(10, 2) |  |  |
| customer_id | BIGINT |  |  |
| customer_name | VARCHAR(255) |  |  |
| id | BIGINT | ✓ |  |

Relates to: customers

