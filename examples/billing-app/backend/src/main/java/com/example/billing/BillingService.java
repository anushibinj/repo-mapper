package com.example.billing;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
public class BillingService {

    @Autowired
    private BillingRepository billingRepository;

    public List<Invoice> findAll() {
        return billingRepository.findAll();
    }

    public Invoice findById(Long id) {
        return billingRepository.findById(id).orElse(null);
    }

    public Invoice create(Invoice invoice) {
        return billingRepository.save(invoice);
    }
}
