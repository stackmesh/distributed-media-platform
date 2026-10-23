package com.stackmesh.service;

import com.stackmesh.model.Account;
import jakarta.persistence.LockModeType;
import java.util.List;
import java.util.Optional;
import java.util.UUID;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Lock;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.transaction.annotation.Transactional;

public interface AccountRepository extends JpaRepository<Account, UUID> {

    @Lock(LockModeType.PESSIMISTIC_WRITE)
    @Query("select a from Account a where a.tenantId = :tenantId and a.code = :code")
    Optional<Account> lockByTenantIdAndCode(@Param("tenantId") String tenantId, @Param("code") String code);

    List<Account> findByTenantIdOrderByCodeAsc(String tenantId);

    @Transactional
    @Modifying(flushAutomatically = true, clearAutomatically = true)
    @Query(
            value =
                    """
                    insert into account (id, tenant_id, code, kind, balance_micros, version)
                    values (:id, :tenantId, 'PREPAID', 'LIABILITY', 0, 0)
                    on conflict (tenant_id, code) do nothing
                    """,
            nativeQuery = true)
    void insertPrepaidIfAbsent(@Param("id") UUID id, @Param("tenantId") String tenantId);
}
