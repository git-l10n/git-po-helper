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

test_expect_success "zh_CN.po: all translated" '
	test_must_fail git -C workdir $HELPER check-po  --pot-file=po/git.pot \
		--report-file-locations=none po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=info msg="[po/ko.po]    3608 translated messages."
------------------------------------------------------------------------------
level=error msg="[po/ko.po]    mismatched patterns: refs/heads"
level=error msg="[po/ko.po]    >> msgid: HEAD not found below refs/heads!"
level=error msg="[po/ko.po]    >> msgstr: 레퍼런스/헤드 아래에 HEAD가 없습니다!"
level=error msg="[po/ko.po]"
level=error msg="[po/ko.po]    mismatched patterns: refs/remotes/"
level=error msg="[po/ko.po]    >> msgid: Note: A branch outside the refs/remotes/ hierarchy was not removed;"
level=error msg="[po/ko.po]    to delete it, use:"
level=error msg="[po/ko.po]    >> msgstr: 알림: 레퍼런스/리모트/ 계층 구조 밖에 있는 일부 브랜치가 제거되지 않았습니다."
level=error msg="[po/ko.po]    삭제하려면 다음을 사용하십시오:"
level=error msg="[po/ko.po]"
level=error msg="[po/ko.po]    mismatched patterns: refs/remotes/<...>/HEAD"
level=error msg="[po/ko.po]    >> msgid: delete refs/remotes/<name>/HEAD"
level=error msg="[po/ko.po]    >> msgstr: 레퍼런스/리모트/<이름>/HEAD 값을 삭제합니다"
level=error msg="[po/ko.po]"
level=error msg="[po/ko.po]    mismatched patterns: refs/remotes/<...>/HEAD"
level=error msg="[po/ko.po]    >> msgid: set refs/remotes/<name>/HEAD according to remote"
level=error msg="[po/ko.po]    >> msgstr: 레퍼런스/리모트/<이름>/HEAD 값을 리모트에 맞게 설정합니다"
level=error msg="[po/ko.po]"
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
ERROR: check-po command failed
EOF

test_expect_success "ko.po: has untranslated strings" '
	test_must_fail git -C workdir $HELPER check-po --pot-file=po/git.pot \
		--report-file-locations=none po/ko.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
