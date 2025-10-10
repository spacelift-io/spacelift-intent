DROP INDEX IF EXISTS idx_operations_provider;
DROP INDEX IF EXISTS idx_operations_type;
DROP INDEX IF EXISTS idx_operations_resource;
DROP INDEX IF EXISTS idx_timeline_operation;
DROP INDEX IF EXISTS idx_timeline_created_at;
DROP INDEX IF EXISTS idx_timeline_resource;
DROP INDEX IF EXISTS idx_dependency_to;
DROP INDEX IF EXISTS idx_dependency_from;
DROP INDEX IF EXISTS idx_state_created_at;
DROP INDEX IF EXISTS idx_provider_type;

DROP TABLE IF EXISTS operations;
DROP TABLE IF EXISTS timeline_events;
DROP TABLE IF EXISTS dependency_edges;
DROP TABLE IF EXISTS state_records;
