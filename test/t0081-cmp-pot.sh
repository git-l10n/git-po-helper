#!/bin/sh

test_description="compare po/git.pot"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	(
		cd workdir &&
		git checkout master &&
		test -f po/git.pot
	)
'

test_expect_success "zh_CN.po: all translated" '
	test_must_fail git -C workdir $HELPER check-po  --pot-file=po/git.pot \
		--report-file-locations=none po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cp "$TEST_DIRECTORY/t0081-zh_CN.expect" expect &&
	test_cmp expect actual
'

test_expect_success "ko.po: has untranslated strings" '
	test_must_fail git -C workdir $HELPER check-po --pot-file=po/git.pot \
		--report-file-locations=none po/ko.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cp "$TEST_DIRECTORY/t0081-ko.expect" expect &&
	test_cmp expect actual
'

test_done
