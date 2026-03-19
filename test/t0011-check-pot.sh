#!/bin/sh

test_description="test git-po-helper check-po on .pot (CamelCase config check)"

. ./lib/test-lib.sh

HELPER="po-helper"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir
'

test_expect_success "prepare pot file" '
	git -C workdir checkout po-2.31.1 &&
	test -f workdir/po/git.pot &&
	sed -e "s|\(Project-Id-Version:\) PACKAGE VERSION|\1Git|" \
		workdir/po/git.pot >workdir/po/git.pot.tmp &&
	mv workdir/po/git.pot.tmp workdir/po/git.pot
'


test_expect_success "check-po on git.pot (Git project, CamelCase check)" '
	test_must_fail git -C workdir po-helper check-po po/git.pot >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cp "$TEST_DIRECTORY/t0011-check-pot.expect" expect &&
	test_cmp expect actual
'

test_done
