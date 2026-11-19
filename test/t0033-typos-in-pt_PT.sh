#!/bin/sh

test_description="check typos in pt_PT.po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn --report-file-locations=none"
POT_NO="--pot-file=no"

test_expect_success "checkout po-2.31.1" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1
'

test_expect_success "check typos in pt_PT.po" '
	test_must_fail git -C workdir $HELPER check-po $POT_NO \
		po/pt_PT.po >out 2>&1 &&
	format_legacy_po_check_output <out >actual &&
	cp "$TEST_DIRECTORY/t0033-typos-in-pt_PT.expect" expect &&
	test_cmp expect actual
'

test_expect_success "only initial-letter typos in master branch" '
	git -C workdir checkout master &&
	test_must_fail git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error \
		po/pt_PT.po >out 2>&1 &&
	test_grep "ERROR.*initial letter case mismatch" out &&
	format_legacy_po_check_output <out >actual &&
	! grep "^ ERROR " actual
'

test_done
