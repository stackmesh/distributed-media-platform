package com.stackmesh.service;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
public record UsageEvent(
        String eventId, String tenantId, String assetId, String meter, long quantity, String occurredAt) {}
