package com.stackmesh.service;

public final class UnknownAccountException extends RuntimeException {

    private final String tenantId;
    private final String code;

    public UnknownAccountException(String tenantId, String code) {
        super("unknown account");
        this.tenantId = tenantId;
        this.code = code;
    }

    public String getTenantId() {
        return tenantId;
    }

    public String getCode() {
        return code;
    }
}
