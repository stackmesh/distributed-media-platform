package com.stackmesh.service;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Component;

@Component
public class UsageListener {

    private static final Logger log = LoggerFactory.getLogger(UsageListener.class);

    private final AccountTransactionService transactions;

    public UsageListener(AccountTransactionService transactions) {
        this.transactions = transactions;
    }

    @KafkaListener(topics = "${stackmesh.billing.topic}")
    public void onUsage(UsageEvent event) {
        Posted posted = transactions.apply(event);
        if (posted.duplicate()) {
            log.info("ignored duplicate usage {}", posted.eventId());
            return;
        }
        log.info("posted usage {} journal {} micros {}", posted.eventId(), posted.journalId(), posted.amountMicros());
    }
}
