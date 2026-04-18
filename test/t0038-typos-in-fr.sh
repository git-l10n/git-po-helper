#!/bin/sh

test_description="check typos in fr.po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn --report-file-locations=none"
POT_NO="--pot-file=no"

test_expect_success "checkout master branch" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout master
'

test_expect_success "still has typos in master branch" '
	test_must_fail git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error po/fr.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cp "$TEST_DIRECTORY/t0038-typos-in-fr.expect" expect &&
	test_cmp expect actual
'

test_done
