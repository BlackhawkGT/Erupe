BEGIN;

ALTER TABLE characters ADD stampcard INT NOT NULL DEFAULT 0;

END;