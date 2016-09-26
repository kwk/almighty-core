-- This table sets the foundation for all future database migrations.
create table "version" (
  "id" serial primary key,
  "updated_at" timestamp with time zone not null default current_timestamp,
  "version" int unique not null
);