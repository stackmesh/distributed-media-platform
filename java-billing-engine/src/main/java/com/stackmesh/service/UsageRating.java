package com.stackmesh.service;

import org.springframework.stereotype.Component;

@Component
public class UsageRating {

    public long priceMicros(String meter, long quantity) {
        if (meter == null) {
            throw new IllegalArgumentException("meter");
        }
        if (quantity <= 0) {
            throw new IllegalArgumentException("quantity");
        }
        return switch (meter) {
            case "storage.bytes" -> ceilKib(quantity) * 2L;
            case "egress.bytes" -> ceilKib(quantity) * 5L;
            case "inference.tokens" -> Math.multiplyExact(quantity, 20L);
            default -> throw new IllegalArgumentException("meter");
        };
    }

    public String revenueCode(String meter) {
        if (meter == null) {
            throw new IllegalArgumentException("meter");
        }
        return switch (meter) {
            case "storage.bytes" -> "REV_STORAGE";
            case "egress.bytes" -> "REV_EGRESS";
            case "inference.tokens" -> "REV_INFERENCE";
            default -> throw new IllegalArgumentException("meter");
        };
    }

    private static long ceilKib(long bytes) {
        if (bytes > Long.MAX_VALUE - 1023L) {
            throw new IllegalArgumentException("quantity");
        }
        return (bytes + 1023L) / 1024L;
    }
}
