package com.stackmesh.service;

public final class InsufficientBalanceException extends RuntimeException {

    private final long available;
    private final long requested;

    public InsufficientBalanceException(long available, long requested) {
        super("insufficient balance");
        this.available = available;
        this.requested = requested;
    }

    public long getAvailable() {
        return available;
    }

    public long getRequested() {
        return requested;
    }
}
