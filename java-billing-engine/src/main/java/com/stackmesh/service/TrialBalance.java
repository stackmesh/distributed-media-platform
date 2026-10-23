package com.stackmesh.service;

public record TrialBalance(long debitMicros, long creditMicros, boolean balanced) {}
