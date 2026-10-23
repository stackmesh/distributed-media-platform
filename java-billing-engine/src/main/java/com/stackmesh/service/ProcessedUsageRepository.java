package com.stackmesh.service;

import com.stackmesh.model.ProcessedUsage;
import org.springframework.data.jpa.repository.JpaRepository;

public interface ProcessedUsageRepository extends JpaRepository<ProcessedUsage, String> {}
