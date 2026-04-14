-- Разделяем window_start/window_end (TIME) на раздельные колонки дата + время,
-- чтобы поддерживать временны́е окна с конкретными датами.
-- Старые значения (только время, без даты) не конвертируются — обнуляются.

ALTER TABLE tasks
    DROP COLUMN window_start,
    DROP COLUMN window_end;

ALTER TABLE tasks
    ADD COLUMN window_start_date DATE NULL,
    ADD COLUMN window_start_time TIME NULL,
    ADD COLUMN window_end_date   DATE NULL,
    ADD COLUMN window_end_time   TIME NULL;
