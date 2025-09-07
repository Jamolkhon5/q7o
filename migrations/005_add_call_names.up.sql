-- Add caller_name and callee_name columns to calls table
ALTER TABLE calls ADD COLUMN caller_name VARCHAR(255);
ALTER TABLE calls ADD COLUMN callee_name VARCHAR(255);

-- Update existing records with user names
UPDATE calls SET 
    caller_name = CONCAT(caller.first_name, ' ', caller.last_name),
    callee_name = CONCAT(callee.first_name, ' ', callee.last_name)
FROM users caller, users callee 
WHERE calls.caller_id = caller.id 
AND calls.callee_id = callee.id;