-- Откат: убираем раздельные колонки, восстанавливаем window_start/window_end как TIME.
ALTER TABLE tasks
    DROP COLUMN window_start_date,
    DROP COLUMN window_start_time,
    DROP COLUMN window_end_date,
    DROP COLUMN window_end_time;

ALTER TABLE tasks
    ADD COLUMN window_start TIME NULL,
    ADD COLUMN window_end   TIME NULL;
