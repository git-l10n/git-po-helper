#!/bin/sh

test_description="check typos in bg.po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn --report-file-locations=none"
POT_NO="--pot-file=no"

test_expect_success "checkout po-2.31.1" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1
'


test_expect_success "check typos in bg.po" '
	git -C workdir $HELPER check-po $POT_NO po/bg.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cp "$TEST_DIRECTORY/t0037-typos-in-bg.expect" expect &&
	test_cmp expect actual
'


test_expect_success "still has typos in master branch" '
	git -C workdir checkout master &&
	test_must_fail git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error po/bg.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cp "$TEST_DIRECTORY/t0037-typos-master.expect" expect &&
	test_cmp expect actual
'

test_done
