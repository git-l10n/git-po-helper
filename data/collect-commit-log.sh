#!/bin/sh

DIR=$(dirname $0)

git log --pretty="%H %s" --merges -- \
	":(exclude)po" \
	":(exclude)git-gui" \
	":(exclude)gitk-git" \
	":(exclude)contrib" \
	>$DIR/all/merge-no-subsystem.txt
git log --pretty="%H %s" --no-merges -- \
	":(exclude)po" \
	":(exclude)git-gui" \
	":(exclude)gitk-git" \
	":(exclude)contrib" \
	>$DIR/all/no-merge-no-subsystem.txt
git log --pretty="%H %s" -- \
	":(exclude)po" \
	":(exclude)git-gui" \
	":(exclude)gitk-git" \
	":(exclude)contrib" \
	>$DIR/all/all-no-subsystem.txt
git log --pretty="%H %s" -- \
	>$DIR/all/all.txt

git log --since "2006-01-01" --pretty="%H %s" --merges -- \
	":(exclude)po" \
	":(exclude)git-gui" \
	":(exclude)gitk-git" \
	":(exclude)contrib" \
	>$DIR/since-2006/merge-no-subsystem.txt
git log --since "2006-01-01" --pretty="%H %s" --no-merges -- \
	":(exclude)po" \
	":(exclude)git-gui" \
	":(exclude)gitk-git" \
	":(exclude)contrib" \
	>$DIR/since-2006/no-merge-no-subsystem.txt
git log --since "2006-01-01" --pretty="%H %s" -- \
	":(exclude)po" \
	":(exclude)git-gui" \
	":(exclude)gitk-git" \
	":(exclude)contrib" \
	>$DIR/since-2006/all-no-subsystem.txt
git log --since "2006-01-01" --pretty="%H %s" -- \
	>$DIR/since-2006/all.txt

