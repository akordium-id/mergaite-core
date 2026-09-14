DROP TABLE IF EXISTS entity_attachments CASCADE;
DROP TABLE IF EXISTS files CASCADE;
DELETE FROM permissions WHERE category = 'file';
