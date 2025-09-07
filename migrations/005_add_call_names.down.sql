-- Remove caller_name and callee_name columns from calls table
ALTER TABLE calls DROP COLUMN caller_name;
ALTER TABLE calls DROP COLUMN callee_name;