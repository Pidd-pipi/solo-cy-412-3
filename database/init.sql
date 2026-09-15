-- SmartEstate MySQL initialization. GORM performs schema migration on startup.
CREATE TABLE IF NOT EXISTS schema_bootstrap (id INT PRIMARY KEY, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
INSERT IGNORE INTO schema_bootstrap (id) VALUES (1);
