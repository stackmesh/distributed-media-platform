package com.stackmesh.service;

import com.stackmesh.model.Account;
import com.stackmesh.model.EntrySide;
import com.stackmesh.model.JournalEntry;
import com.stackmesh.model.LedgerLine;
import com.stackmesh.model.ProcessedUsage;
import java.time.Instant;
import java.util.List;
import java.util.UUID;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class AccountTransactionService {

    private final AccountRepository accounts;
    private final JournalRepository journals;
    private final LedgerLineRepository lines;
    private final ProcessedUsageRepository processed;
    private final UsageRating rating;

    public AccountTransactionService(
            AccountRepository accounts,
            JournalRepository journals,
            LedgerLineRepository lines,
            ProcessedUsageRepository processed,
            UsageRating rating) {
        this.accounts = accounts;
        this.journals = journals;
        this.lines = lines;
        this.processed = processed;
        this.rating = rating;
    }

    @Transactional
    public Posted apply(UsageEvent event) {
        if (event == null) {
            throw new IllegalArgumentException("event");
        }
        if (!TenantIds.valid(event.tenantId())) {
            throw new IllegalArgumentException("tenantId");
        }
        long amount = rating.priceMicros(event.meter(), event.quantity());
        return post(
                event.eventId(),
                event.meter(),
                new AccountRef(event.tenantId(), "PREPAID"),
                new AccountRef("platform", rating.revenueCode(event.meter())),
                amount,
                occurred(event.occurredAt()));
    }

    @Transactional
    public Posted grant(String eventId, String tenantId, long amount) {
        if (!TenantIds.valid(tenantId)) {
            throw new IllegalArgumentException("tenantId");
        }
        if (amount <= 0) {
            throw new IllegalArgumentException("amount");
        }
        accounts.insertPrepaidIfAbsent(UUID.randomUUID(), tenantId);
        return post(
                eventId,
                "prepaid.grant",
                new AccountRef("platform", "PROMO_EXPENSE"),
                new AccountRef(tenantId, "PREPAID"),
                amount,
                Instant.now());
    }

    @Transactional(readOnly = true)
    public TrialBalance trial() {
        long debit = lines.sumBySide(EntrySide.DEBIT);
        long credit = lines.sumBySide(EntrySide.CREDIT);
        return new TrialBalance(debit, credit, debit == credit);
    }

    @Transactional(readOnly = true)
    public List<AccountView> accounts(String tenantId) {
        return accounts.findByTenantIdOrderByCodeAsc(tenantId).stream()
                .map(AccountView::from)
                .toList();
    }

    private Posted post(
            String eventId,
            String memo,
            AccountRef debit,
            AccountRef credit,
            long amount,
            Instant occurredAt) {
        if (eventId == null || eventId.isBlank() || eventId.length() > 160) {
            throw new IllegalArgumentException("eventId");
        }
        if (memo == null || memo.isBlank() || memo.length() > 128) {
            throw new IllegalArgumentException("memo");
        }
        if (debit.equals(credit)) {
            throw new IllegalArgumentException("debit and credit accounts must differ");
        }
        Posting.requireBalanced(amount, amount);

        var prior = processed.findById(eventId);
        if (prior.isPresent()) {
            long original = lines.sumByEntryAndSide(prior.get().getJournalId(), EntrySide.DEBIT);
            return new Posted(prior.get().getJournalId(), eventId, original, true);
        }

        AccountRef first = debit.rank().compareTo(credit.rank()) <= 0 ? debit : credit;
        AccountRef second = first.equals(debit) ? credit : debit;
        Account lockedFirst = require(first);
        Account lockedSecond = require(second);
        Account debitAccount = matches(lockedFirst, debit) ? lockedFirst : lockedSecond;
        Account creditAccount = debitAccount == lockedFirst ? lockedSecond : lockedFirst;

        debitAccount.setBalanceMicros(Posting.applyBalance(
                debitAccount.getKind(), debitAccount.getBalanceMicros(), EntrySide.DEBIT, amount));
        creditAccount.setBalanceMicros(Posting.applyBalance(
                creditAccount.getKind(), creditAccount.getBalanceMicros(), EntrySide.CREDIT, amount));

        Instant createdAt = Instant.now();
        JournalEntry entry = new JournalEntry();
        entry.setId(UUID.randomUUID());
        entry.setEventId(eventId);
        entry.setMemo(memo);
        entry.setOccurredAt(occurredAt);
        entry.setCreatedAt(createdAt);
        journals.save(entry);

        lines.save(line(entry.getId(), debitAccount.getId(), EntrySide.DEBIT, amount));
        lines.save(line(entry.getId(), creditAccount.getId(), EntrySide.CREDIT, amount));

        ProcessedUsage done = new ProcessedUsage();
        done.setEventId(eventId);
        done.setJournalId(entry.getId());
        done.setProcessedAt(createdAt);
        processed.save(done);
        return new Posted(entry.getId(), eventId, amount, false);
    }

    private Account require(AccountRef ref) {
        return accounts
                .lockByTenantIdAndCode(ref.tenant(), ref.code())
                .orElseThrow(() -> new UnknownAccountException(ref.tenant(), ref.code()));
    }

    private static boolean matches(Account account, AccountRef ref) {
        return account.getTenantId().equals(ref.tenant()) && account.getCode().equals(ref.code());
    }

    private static LedgerLine line(UUID entryId, UUID accountId, EntrySide side, long amount) {
        LedgerLine line = new LedgerLine();
        line.setId(UUID.randomUUID());
        line.setEntryId(entryId);
        line.setAccountId(accountId);
        line.setSide(side);
        line.setAmountMicros(amount);
        return line;
    }

    private static Instant occurred(String raw) {
        if (raw == null || raw.isBlank()) {
            return Instant.now();
        }
        try {
            return Instant.parse(raw);
        } catch (RuntimeException ex) {
            throw new IllegalArgumentException("occurredAt");
        }
    }

    private record AccountRef(String tenant, String code) {
        private String rank() {
            return tenant + "\0" + code;
        }
    }
}
