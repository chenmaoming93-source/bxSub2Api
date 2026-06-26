-- Add group supported model scopes.
ALTER TABLE `groups`
ADD COLUMN supported_model_scopes JSON NULL;