#!/bin/sh

# Create toplevel gitdir to prevent dangous operation on current repo.
git -c init.defaultBranch=master init -q "$SHARNESS_TRASH_DIRECTORY"
