package com.stackmesh.service;

/** Tenant ids match the gateway: 1-63 chars, lowercase letters, digits, interior hyphens. */
public final class TenantIds {

    private TenantIds() {}

    public static boolean valid(String tenantId) {
        if (tenantId == null || tenantId.length() < 1 || tenantId.length() > 63) {
            return false;
        }
        for (int i = 0; i < tenantId.length(); i++) {
            char c = tenantId.charAt(i);
            boolean alnum = (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9');
            if (alnum) {
                continue;
            }
            if (c == '-' && i > 0 && i < tenantId.length() - 1) {
                continue;
            }
            return false;
        }
        return true;
    }
}
