-- Revert 000026: remove the employee.update_self permission.
--
-- role_permissions rows are removed first so the foreign key does not block
-- the permissions delete.

DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE name = 'employee.update_self');

DELETE FROM permissions WHERE name = 'employee.update_self';
