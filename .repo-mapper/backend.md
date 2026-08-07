# Backend

## Controllers

| Name | File | Depends On |
|---|---|---|
| AuthController | examples/billing-app/backend/src/main/java/com/example/billing/AuthController.java | authService |
| BillingController | examples/billing-app/backend/src/main/java/com/example/billing/BillingController.java | billingService |

## Services

| Name | File | Depends On |
|---|---|---|
| AuthService | examples/billing-app/backend/src/main/java/com/example/billing/AuthService.java |  |
| BillingService | examples/billing-app/backend/src/main/java/com/example/billing/BillingService.java | billingRepository |

## Repositories

| Name | File | Depends On |
|---|---|---|
| BillingRepository | examples/billing-app/backend/src/main/java/com/example/billing/BillingRepository.java |  |

## Entities

| Name | File | Depends On |
|---|---|---|
| Customer | examples/billing-app/backend/src/main/java/com/example/billing/Customer.java |  |
| Invoice | examples/billing-app/backend/src/main/java/com/example/billing/Invoice.java |  |

## API Endpoints

| Method | Path | Controller | Handler |
|---|---|---|---|
| POST | /auth/login | AuthController | login |
| GET | /billing/invoices | BillingController | listInvoices |
| POST | /billing/invoices | BillingController | createInvoice |
| GET | /billing/invoices/{id} | BillingController | getInvoice |

