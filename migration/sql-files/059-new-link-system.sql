CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- fillfactor (integer)
-- 
-- The fillfactor for a table is a percentage between 10 and 100. 100 (complete
-- packing) is the default. When a smaller fillfactor is specified, INSERT
-- operations pack table pages only to the indicated percentage; the remaining
-- space on each page is reserved for updating rows on that page. This gives UPDATE
-- a chance to place the updated copy of a row on the same page as the original,
-- which is more efficient than placing it on a different page. For a table whose
-- entries are never updated, complete packing is the best choice, but in heavily
-- updated tables smaller fillfactors are appropriate. This parameter cannot be set
-- for TOAST tables.

CREATE TABLE work_item_link_matrix (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    version integer,
    type_id uuid,
    source_type_id uuid REFERENCES work_item_types(id) ON DELETE CASCADE,
    target_type_id uuid REFERENCES work_item_types(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX work_item_link_matrix_uniq ON work_item_link_matrix (type_id, source_type_id, target_type_id)
    WHERE deleted_at IS NULL;

-- Let's make this migration really straight forward and don't handle complex
-- cases that simply don't exist. Simply remove source and target columns from
-- the link types table.
ALTER TABLE work_item_link_types DROP COLUMN source_type_id;
ALTER TABLE work_item_link_types DROP COLUMN target_type_id;

-- Existing links don't have to be migrated at all because the just reference
-- the link types table which is still in place.

-- Transfer all link types to the matrix system
-- INSERT INTO work_item_link_matrix (type_id, source_type_id, target_type_id)
--    SELECT id, source_type_id, target_type_id FROM work_item_link_types WHERE forward_name <> 'parent of';
