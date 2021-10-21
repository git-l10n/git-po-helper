#!/bin/sh

test_description="test git-po-helper version"

. ./lib/sharness.sh

HELPER="po-helper --no-gettext-back-compatible"

test_expect_success "git-po-helper version output test" '
	git $HELPER version >out &&
	grep "^git-po-helper version" out >expect &&
	test -s expect
'

test_expect_success "check git-po-helper version format" '
	grep "^git-po-helper version [0-9]\+\.[0-9]\+\.[0-9]\+" out >actual &&
	test_cmp expect actual
'

test_done
