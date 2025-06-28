
ALTER TABLE scoring_presets DROP CONSTRAINT IF EXISTS scoring_presets_event_id_fkey;
ALTER TABLE scoring_presets
ADD CONSTRAINT scoring_presets_event_id_fkey FOREIGN KEY (event_id) REFERENCES bpl2.events(id) ON DELETE CASCADE;