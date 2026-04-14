UPDATE tasks SET duration_min = 0 WHERE duration_min IS NULL;
ALTER TABLE tasks ALTER COLUMN duration_min SET NOT NULL;
