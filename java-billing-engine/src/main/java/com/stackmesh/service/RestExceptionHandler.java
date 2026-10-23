package com.stackmesh.service;

import java.util.Map;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.MethodArgumentNotValidException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;

@RestControllerAdvice
public class RestExceptionHandler {

    @ExceptionHandler(InsufficientBalanceException.class)
    ResponseEntity<Map<String, Object>> insufficient(InsufficientBalanceException ex) {
        return ResponseEntity.status(HttpStatus.CONFLICT)
                .body(Map.of(
                        "error", "insufficient_balance",
                        "availableMicros", ex.getAvailable(),
                        "requestedMicros", ex.getRequested()));
    }

    @ExceptionHandler(UnknownAccountException.class)
    ResponseEntity<Map<String, String>> unknown(UnknownAccountException ex) {
        return ResponseEntity.status(HttpStatus.NOT_FOUND)
                .body(Map.of("error", "unknown_account", "account", ex.getTenantId() + "/" + ex.getCode()));
    }

    @ExceptionHandler({IllegalArgumentException.class, MethodArgumentNotValidException.class})
    ResponseEntity<Map<String, String>> invalid(Exception ex) {
        return ResponseEntity.badRequest().body(Map.of("error", "invalid_request"));
    }
}
