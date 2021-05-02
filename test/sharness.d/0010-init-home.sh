#!/bin/sh

# Create toplevel gitdir to prevent dangous operation on current repo.
git init "$SHARNESS_TRASH_DIRECTORY"
