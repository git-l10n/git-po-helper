#!/bin/sh

test_description="check file-locations in po file"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions"
POT_NO="--pot-file=no"

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
------------------------------------------------------------------------------
level=error msg="[po/zh_CN.po]    Found file-location comments in po file. By submitting a location-less"
level=error msg="[po/zh_CN.po]    \"po/XX.po\" file, the size of the Git repository can be greatly reduced."
level=error msg="[po/zh_CN.po]    See the discussion below:"
level=error msg="[po/zh_CN.po]"
level=error msg="[po/zh_CN.po]     https://lore.kernel.org/git/20220504124121.12683-1-worldhello.net@gmail.com/"
level=error msg="[po/zh_CN.po]"
level=error msg="[po/zh_CN.po]    As how to commit a location-less \"po/XX.po\" file, See:"
level=error msg="[po/zh_CN.po]"
level=error msg="[po/zh_CN.po]     the [Updating a \"XX.po\" file] section in \"po/README.md\""
------------------------------------------------------------------------------
level=error msg="[po/zh_CN.po]    mismatched patterns: refs/remotes/"
level=error msg="[po/zh_CN.po]    >> msgid: Note: A branch outside the refs/remotes/ hierarchy was not removed;"
level=error msg="[po/zh_CN.po]    to delete it, use:"
level=error msg="[po/zh_CN.po]    >> msgstr: 注意：ref/remotes 层级之外的一个分支未被移除。要删除它，使用："
level=error msg="[po/zh_CN.po]"
level=error msg="[po/zh_CN.po]    mismatched patterns: refs/remotes/"
level=error msg="[po/zh_CN.po]    >> msgid: Note: Some branches outside the refs/remotes/ hierarchy were not removed;"
level=error msg="[po/zh_CN.po]    to delete them, use:"
level=error msg="[po/zh_CN.po]    >> msgstr: 注意：ref/remotes 层级之外的一些分支未被移除。要删除它们，使用："
level=error msg="[po/zh_CN.po]"
ERROR: check-po command failed
EOF

test_expect_success "zh_CN.po: has file-locations (--report-file-location=error)" '
	test_must_fail git -C workdir $HELPER check-po $POT_NO \
		--report-file-locations=error po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "zh_CN.po: has file-locations (no --report-file-location option)" '
	test_must_fail git -C workdir $HELPER check-po $POT_NO \
		po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=info msg="[po/zh_CN.po]    5282 translated messages."
------------------------------------------------------------------------------
level=error msg="[po/zh_CN.po]    mismatched patterns: refs/remotes/"
level=error msg="[po/zh_CN.po]    >> msgid: Note: A branch outside the refs/remotes/ hierarchy was not removed;"
level=error msg="[po/zh_CN.po]    to delete it, use:"
level=error msg="[po/zh_CN.po]    >> msgstr: 注意：ref/remotes 层级之外的一个分支未被移除。要删除它，使用："
level=error msg="[po/zh_CN.po]"
level=error msg="[po/zh_CN.po]    mismatched patterns: refs/remotes/"
level=error msg="[po/zh_CN.po]    >> msgid: Note: Some branches outside the refs/remotes/ hierarchy were not removed;"
level=error msg="[po/zh_CN.po]    to delete them, use:"
level=error msg="[po/zh_CN.po]    >> msgstr: 注意：ref/remotes 层级之外的一些分支未被移除。要删除它们，使用："
level=error msg="[po/zh_CN.po]"
ERROR: check-po command failed
EOF

test_expect_success "zh_CN.po: remove locations" '
	(
		cd workdir &&
		msgcat --add-location=file po/zh_CN.po -o po/zh_CN.poX &&
		mv po/zh_CN.poX po/zh_CN.po
	) &&
	test_must_fail git -C workdir $HELPER check-po $POT_NO \
		po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "zh_CN.po: remove both files and locations" '
	(
		cd workdir &&
		msgcat --no-location po/zh_CN.po -o po/zh_CN.poX &&
		mv po/zh_CN.poX po/zh_CN.po
	) &&
	test_must_fail git -C workdir $HELPER check-po $POT_NO \
		po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
