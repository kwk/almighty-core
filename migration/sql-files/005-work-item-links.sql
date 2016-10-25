-- Here's the layout I'm trying to create:
-- (NOTE: work_items and work_item_types tables already exist)
--        
--           .----------------.
--           | work_items     |        .-----------------.
--           | ----------     |        | work_item_types |
--     .------>id bigserial   |        | --------------- |
--     |     | type text ------------>>| name text       |
--     |     | [other fields] |    |   | [other fields]  |
--     |     '----------------'    |   '-----------------'
--     |                           |
--     |    .-----------------.    |   .---------------------.
--     |    | work_item_links |    |   | work_item_link_types|
--     |    | --------------- |    |   | ---------           |
--     |    | id uuid         | .------> id uuid             |
--     .------source bigint   | |  |   | name text           |
--      '-----target bigint   |/   |   | description text    |
--          | type uuid ------'    ------source_type text    |
--          | comment text    |     '----target_type text    |
--          '-----------------'        | forward_name text   |
--                                     | reverse_name text   |
--    .--------------------------.    .- category uuid       |
--    |work_item_link_categories |   / '---------------------'
--    |------------------------- |  /
--    | id uuid                 <---
--    | name text                |
--    | description text         |
--    '--------------------------'

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- work item link categories

CREATE TABLE work_item_link_categories (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone DEFAULT NULL,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    name text NOT NULL UNIQUE,
    description text,
    version integer
);

-- work item link types

CREATE TABLE work_item_link_types (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone DEFAULT NULL,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    name text NOT NULL,
    description text,
    source_type text REFERENCES work_item_types(name) NOT NULL,
    target_type text REFERENCES work_item_types(name) NOT NULL,
    forward_name text NOT NULL, -- MUST not be NULL because UI needs this
    reverse_name text NOT NULL, -- MUST not be NULL because UI needs this
    link_category uuid REFERENCES work_item_link_categories(id),
    version integer
);

-- work item links

CREATE TABLE work_item_links (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone DEFAULT NULL,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    type uuid REFERENCES work_item_link_types(id) NOT NULL,
    source bigint REFERENCES work_items(id) NOT NULL,
    target bigint REFERENCES work_items(id) NOT NULL,
    comment text DEFAULT NULL,
    version integer
);