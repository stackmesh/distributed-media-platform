package com.stackmesh.service;

import com.stackmesh.model.AccountKind;
import com.stackmesh.model.EntrySide;

public final class Posting {

    private Posting() {}

    public static void requireBalanced(long debit, long credit) {
        if (debit != credit || debit <= 0) {
            throw new IllegalArgumentException("unbalanced posting");
        }
    }

    public static long applyBalance(AccountKind kind, long balance, EntrySide side, long amount) {
        if (kind == null || side == null) {
            throw new IllegalArgumentException("posting");
        }
        if (amount <= 0) {
            throw new IllegalArgumentException("amount");
        }
        boolean debitNormal = kind == AccountKind.ASSET || kind == AccountKind.EXPENSE;
        boolean increase = (debitNormal && side == EntrySide.DEBIT) || (!debitNormal && side == EntrySide.CREDIT);
        long next;
        try {
            next = increase ? Math.addExact(balance, amount) : Math.subtractExact(balance, amount);
        } catch (ArithmeticException ex) {
            throw new IllegalArgumentException("amount overflow");
        }
        if (next < 0) {
            throw new InsufficientBalanceException(balance, amount);
        }
        return next;
    }
}
