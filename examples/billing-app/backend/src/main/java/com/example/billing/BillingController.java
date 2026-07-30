package com.example.billing;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/billing")
public class BillingController {

    @Autowired
    private BillingService billingService;

    @GetMapping("/invoices")
    public List<Invoice> listInvoices() {
        return billingService.findAll();
    }

    @GetMapping("/invoices/{id}")
    public Invoice getInvoice(@PathVariable Long id) {
        return billingService.findById(id);
    }

    @PostMapping("/invoices")
    public Invoice createInvoice(@RequestBody Invoice invoice) {
        return billingService.create(invoice);
    }
}

// test comment for update detection

// another test comment
