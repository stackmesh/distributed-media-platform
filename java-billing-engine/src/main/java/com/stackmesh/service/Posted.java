package com.stackmesh.service;

import java.util.UUID;

public record Posted(UUID journalId, String eventId, long amountMicros, boolean duplicate) {}
