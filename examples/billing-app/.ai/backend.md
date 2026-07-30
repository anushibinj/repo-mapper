# Backend

## Controllers

| Name | File | Depends On |
|---|---|---|
| AuthController | backend/src/main/java/com/example/billing/AuthController.java | authService |
| BillingController | backend/src/main/java/com/example/billing/BillingController.java | billingService |

## Services

| Name | File | Depends On |
|---|---|---|
| AuthService | backend/src/main/java/com/example/billing/AuthService.java |  |
| BillingService | backend/src/main/java/com/example/billing/BillingService.java | billingRepository |

## Repositories

| Name | File | Depends On |
|---|---|---|
| BillingRepository | backend/src/main/java/com/example/billing/BillingRepository.java |  |

## Entities

| Name | File | Depends On |
|---|---|---|
| Customer | backend/src/main/java/com/example/billing/Customer.java |  |
| Invoice | backend/src/main/java/com/example/billing/Invoice.java |  |

## API Endpoints

| Method | Path | Controller | Handler |
|---|---|---|---|
| POST | /auth/login | AuthController | login |
| GET | /billing/invoices | BillingController | listInvoices |
| POST | /billing/invoices | BillingController | createInvoice |
| GET | /billing/invoices/{id} | BillingController | getInvoice |

