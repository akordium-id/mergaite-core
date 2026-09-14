DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS comments CASCADE;
DELETE FROM permissions WHERE category = 'communication';
