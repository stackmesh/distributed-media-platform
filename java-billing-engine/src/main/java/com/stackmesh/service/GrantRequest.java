package com.stackmesh.service;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Positive;
import jakarta.validation.constraints.Size;

public record GrantRequest(
        @NotBlank @Size(max = 160) String eventId,
        @NotBlank @Size(max = 63) String tenantId,
        @Positive long amountMicros) {}
