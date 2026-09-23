-- Revert 000027.
--
-- Nothing to undo: 000027 is a defensive re-assertion of the same rows that
-- 000026 owns. Dropping them here would make a rollback of 000026 alone
-- inconsistent, so the down migration deliberately leaves the data in place.

SELECT 1;
