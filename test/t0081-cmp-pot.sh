#!/bin/sh

test_description="compare po/git.pot"

. ./lib/sharness.sh

HELPER="po-helper --no-special-gettext-versions"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	(
		cd workdir &&
		git checkout master &&
		test -f po/git.pot
	)
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=info msg="[po/zh_CN.po]    5282 translated messages."
EOF

test_expect_success "zh_CN.po: all translated" '
	git -C workdir $HELPER check-po  --pot-file=po/git.pot \
		--report-file-locations=none po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=info msg="[po/ko.po]    3608 translated messages."
------------------------------------------------------------------------------
level=error msg="[po/ko.po]    2242 new string(s) in 'po/git.pot', but not in your 'po/XX.po'"
level=error msg="[po/ko.po]"
level=error msg="[po/ko.po]     > po/git.pot:24: this message is used but not defined in po/ko.po"
level=error msg="[po/ko.po]     > po/git.pot:54: this message is used but not defined in po/ko.po"
level=error msg="[po/ko.po]     > po/git.pot:84: this message is used but not defined in po/ko.po"
level=error msg="[po/ko.po]     > ..."
level=error msg="[po/ko.po]"
level=error msg="[po/ko.po]    568 obsolete string(s) in your 'po/XX.po', which must be removed"
level=error msg="[po/ko.po]"
level=error msg="[po/ko.po]     > po/XX.po:147: warning: this message is not used"
level=error msg="[po/ko.po]     > po/XX.po:172: warning: this message is not used"
level=error msg="[po/ko.po]     > po/XX.po:176: warning: this message is not used"
level=error msg="[po/ko.po]     > ..."
level=error msg="[po/ko.po]"
level=error msg="[po/ko.po]    Please run \"git-po-helper update po/XX.po\" to update your po file,"
level=error msg="[po/ko.po]    and translate the new strings in it."
level=error msg="[po/ko.po]"

ERROR: fail to execute "git-po-helper check-po"
EOF

test_expect_success "ko.po: has untranslated strings" '
	test_must_fail git -C workdir $HELPER check-po --pot-file=po/git.pot \
		--report-file-locations=none po/ko.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
