package com.example.billing;

import javax.persistence.Column;
import javax.persistence.Entity;
import javax.persistence.Id;
import javax.persistence.Table;
import java.math.BigDecimal;

@Entity
@Table(name = "invoices")
public class Invoice {

    @Id
    private Long id;

    @Column(name = "customer_name")
    private String customerName;

    @Column(name = "amount")
    private BigDecimal amount;

    private Customer customer;
}
