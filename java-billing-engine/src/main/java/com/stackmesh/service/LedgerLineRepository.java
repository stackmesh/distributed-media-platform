package com.stackmesh.service;

import com.stackmesh.model.EntrySide;
import com.stackmesh.model.LedgerLine;
import java.util.UUID;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;

public interface LedgerLineRepository extends JpaRepository<LedgerLine, UUID> {

    @Query("select coalesce(sum(l.amountMicros), 0) from LedgerLine l where l.side = :side")
    long sumBySide(@Param("side") EntrySide side);

    @Query(
            "select coalesce(sum(l.amountMicros), 0) from LedgerLine l where l.entryId = :entryId and l.side = :side")
    long sumByEntryAndSide(@Param("entryId") UUID entryId, @Param("side") EntrySide side);
}
