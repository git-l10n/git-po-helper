#!/bin/sh

test_description="test git-po-helper version"

. ./lib/sharness.sh

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	test -f workdir/po/git.pot
'

test_expect_success "git-po-helper version output test" '
	(
		cd workdir &&
		git-po-helper version >out &&
		grep "^git-po-helper version" out >expect &&
		test -s expect
	)
'

test_expect_failure "check git-po-helper version format" '
	(
		cd workdir &&
		grep "^git-po-helper version [0-9]\+\.[0-9]\+\.[0-9]\+" out >actual &&
		test_cmp expect actual
	)
'

test_done
