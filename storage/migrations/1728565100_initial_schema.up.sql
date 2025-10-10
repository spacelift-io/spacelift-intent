-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- State records table
CREATE TABLE IF NOT EXISTS state_records (
	id TEXT PRIMARY KEY,
	provider TEXT NOT NULL,
	version TEXT NOT NULL,
	resource_type TEXT NOT NULL,
	state TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Dependency edges table
CREATE TABLE IF NOT EXISTS dependency_edges (
	from_resource_id TEXT NOT NULL,
	to_resource_id TEXT NOT NULL,
	dependency_type TEXT NOT NULL,
	explanation TEXT DEFAULT '',
	field_mappings TEXT DEFAULT '[]',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (from_resource_id, to_resource_id),
	FOREIGN KEY (from_resource_id) REFERENCES state_records(id) ON DELETE CASCADE,
	FOREIGN KEY (to_resource_id) REFERENCES state_records(id) ON DELETE CASCADE
);

-- Timeline events table
CREATE TABLE IF NOT EXISTS timeline_events (
	id TEXT PRIMARY KEY,
	resource_id TEXT,
	operation TEXT NOT NULL,
	changed_by TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Operations table
CREATE TABLE IF NOT EXISTS operations (
	id TEXT PRIMARY KEY,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	allow TEXT DEFAULT '',
	deny TEXT DEFAULT '',
	failed TEXT DEFAULT '',
	resource_id TEXT NOT NULL,
	resource_type TEXT NOT NULL,
	provider TEXT NOT NULL,
	operation TEXT NOT NULL,
	current_state TEXT DEFAULT '',
	proposed_state TEXT DEFAULT ''
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_provider_type ON state_records(provider, resource_type);
CREATE INDEX IF NOT EXISTS idx_state_created_at ON state_records(created_at);
CREATE INDEX IF NOT EXISTS idx_dependency_from ON dependency_edges(from_resource_id);
CREATE INDEX IF NOT EXISTS idx_dependency_to ON dependency_edges(to_resource_id);
CREATE INDEX IF NOT EXISTS idx_timeline_resource ON timeline_events(resource_id);
CREATE INDEX IF NOT EXISTS idx_timeline_created_at ON timeline_events(created_at);
CREATE INDEX IF NOT EXISTS idx_timeline_operation ON timeline_events(operation);
CREATE INDEX IF NOT EXISTS idx_operations_resource ON operations(resource_id);
CREATE INDEX IF NOT EXISTS idx_operations_type ON operations(resource_type);
CREATE INDEX IF NOT EXISTS idx_operations_provider ON operations(provider);
