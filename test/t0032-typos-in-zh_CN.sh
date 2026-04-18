#!/bin/sh

test_description="check typos in zh_CN.po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn --report-file-locations=none"
POT_NO="--pot-file=no"

test_expect_success "checkout po-2.31.1" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1
'

test_expect_success "check typos in zh_CN.po" '
	git -C workdir $HELPER check-po $POT_NO \
		--report-file-locations=none po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cp "$TEST_DIRECTORY/t0032-typos-zh_CN-po231.expect" expect &&
	test_cmp expect actual
'

test_expect_success "check typos in master branch" '
	git -C workdir checkout master &&
	git -C workdir $HELPER \
		check-po $POT_NO --report-typos=warn \
		--report-file-locations=warn po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cp "$TEST_DIRECTORY/t0032-typos-zh_CN-master.expect" expect &&
	test_cmp expect actual
'

test_done
