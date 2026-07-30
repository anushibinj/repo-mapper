package com.example.billing;

import org.springframework.stereotype.Service;

@Service
public class AuthService {

    public String authenticate(String credentials) {
        return "token";
    }
}
