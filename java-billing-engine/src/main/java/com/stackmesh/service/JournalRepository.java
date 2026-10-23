package com.stackmesh.service;

import com.stackmesh.model.JournalEntry;
import java.util.UUID;
import org.springframework.data.jpa.repository.JpaRepository;

public interface JournalRepository extends JpaRepository<JournalEntry, UUID> {}
