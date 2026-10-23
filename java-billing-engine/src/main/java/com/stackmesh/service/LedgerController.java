package com.stackmesh.service;

import jakarta.validation.Valid;
import java.util.List;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/v1/ledger")
public class LedgerController {

    private final AccountTransactionService ledger;

    public LedgerController(AccountTransactionService ledger) {
        this.ledger = ledger;
    }

    @GetMapping("/trial")
    public TrialBalance trial() {
        return ledger.trial();
    }

    @GetMapping("/accounts/{tenantId}")
    public List<AccountView> accounts(@PathVariable String tenantId) {
        return ledger.accounts(tenantId);
    }

    @PostMapping("/grants")
    public ResponseEntity<Posted> grant(@Valid @RequestBody GrantRequest request) {
        Posted posted = ledger.grant(request.eventId(), request.tenantId(), request.amountMicros());
        HttpStatus status = posted.duplicate() ? HttpStatus.OK : HttpStatus.CREATED;
        return ResponseEntity.status(status).body(posted);
    }
}
