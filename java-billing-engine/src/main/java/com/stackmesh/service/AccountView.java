package com.stackmesh.service;

import com.stackmesh.model.Account;
import com.stackmesh.model.AccountKind;
import java.util.UUID;

public record AccountView(
        UUID id, String tenantId, String code, AccountKind kind, long balanceMicros, long version) {

    public static AccountView from(Account account) {
        return new AccountView(
                account.getId(),
                account.getTenantId(),
                account.getCode(),
                account.getKind(),
                account.getBalanceMicros(),
                account.getVersion());
    }
}
