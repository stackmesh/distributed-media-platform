package com.stackmesh.service;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import com.stackmesh.model.AccountKind;
import com.stackmesh.model.EntrySide;
import org.junit.jupiter.api.Test;

class PostingTest {

    private final UsageRating rating = new UsageRating();

    @Test
    void storageRoundsUpToAKibibyte() {
        assertEquals(2L, rating.priceMicros("storage.bytes", 1));
        assertEquals(2L, rating.priceMicros("storage.bytes", 1024));
        assertEquals(4L, rating.priceMicros("storage.bytes", 1025));
        assertEquals(5L, rating.priceMicros("egress.bytes", 1));
        assertEquals(200L, rating.priceMicros("inference.tokens", 10));
    }

    @Test
    void revenueCodesFollowMeters() {
        assertEquals("REV_STORAGE", rating.revenueCode("storage.bytes"));
        assertEquals("REV_EGRESS", rating.revenueCode("egress.bytes"));
        assertEquals("REV_INFERENCE", rating.revenueCode("inference.tokens"));
        assertThrows(IllegalArgumentException.class, () -> rating.revenueCode("storage.objects"));
        assertThrows(IllegalArgumentException.class, () -> rating.priceMicros("storage.bytes", 0));
    }

    @Test
    void normalBalancesMoveWithThePostingSide() {
        assertEquals(15L, Posting.applyBalance(AccountKind.EXPENSE, 10, EntrySide.DEBIT, 5));
        assertEquals(15L, Posting.applyBalance(AccountKind.REVENUE, 10, EntrySide.CREDIT, 5));
        assertEquals(5L, Posting.applyBalance(AccountKind.LIABILITY, 10, EntrySide.DEBIT, 5));
    }

    @Test
    void overdraftAndUnbalancedPostingsFail() {
        assertThrows(
                InsufficientBalanceException.class,
                () -> Posting.applyBalance(AccountKind.LIABILITY, 4, EntrySide.DEBIT, 5));
        assertThrows(IllegalArgumentException.class, () -> Posting.requireBalanced(1, 2));
        assertThrows(IllegalArgumentException.class, () -> Posting.requireBalanced(0, 0));
    }

    @Test
    void tenantIdsMatchTheGateway() {
        assertEquals(true, TenantIds.valid("demo"));
        assertEquals(true, TenantIds.valid("a"));
        assertEquals(true, TenantIds.valid("studio-east"));
        assertEquals(false, TenantIds.valid("Demo"));
        assertEquals(false, TenantIds.valid("-demo"));
        assertEquals(false, TenantIds.valid("demo-"));
        assertEquals(false, TenantIds.valid(""));
    }
}
