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
ℹ️ Syntax check with msgfmt
 INFO [zh_CN.po] 5282 translated messages.
❌ msgid/msgstr pattern check
 ERROR [zh_CN.po] mismatched patterns: refs/remotes/
 ERROR [zh_CN.po] >> msgid: Note: A branch outside the refs/remotes/ hierarchy was not removed;
 ERROR [zh_CN.po] to delete it, use:
 ERROR [zh_CN.po] >> msgstr: 注意：ref/remotes 层级之外的一个分支未被移除。要删除它，使用：
 ERROR [zh_CN.po]
 ERROR [zh_CN.po] mismatched patterns: refs/remotes/
 ERROR [zh_CN.po] >> msgid: Note: Some branches outside the refs/remotes/ hierarchy were not removed;
 ERROR [zh_CN.po] to delete them, use:
 ERROR [zh_CN.po] >> msgstr: 注意：ref/remotes 层级之外的一些分支未被移除。要删除它们，使用：
 ERROR [zh_CN.po]
ERROR: check-po command failed
EOF

test_expect_success "zh_CN.po: all translated" '
	test_must_fail git -C workdir $HELPER check-po  --pot-file=po/git.pot \
		--report-file-locations=none po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
ℹ️ Syntax check with msgfmt
 INFO [ko.po] 3608 translated messages.
❌ msgid/msgstr pattern check
 ERROR [ko.po] mismatched patterns: refs/heads
 ERROR [ko.po] >> msgid: HEAD not found below refs/heads!
 ERROR [ko.po] >> msgstr: 레퍼런스/헤드 아래에 HEAD가 없습니다!
 ERROR [ko.po]
 ERROR [ko.po] mismatched patterns: refs/remotes/
 ERROR [ko.po] >> msgid: Note: A branch outside the refs/remotes/ hierarchy was not removed;
 ERROR [ko.po] to delete it, use:
 ERROR [ko.po] >> msgstr: 알림: 레퍼런스/리모트/ 계층 구조 밖에 있는 일부 브랜치가 제거되지 않았습니다.
 ERROR [ko.po] 삭제하려면 다음을 사용하십시오:
 ERROR [ko.po]
 ERROR [ko.po] mismatched patterns: refs/remotes/<...>/HEAD
 ERROR [ko.po] >> msgid: delete refs/remotes/<name>/HEAD
 ERROR [ko.po] >> msgstr: 레퍼런스/리모트/<이름>/HEAD 값을 삭제합니다
 ERROR [ko.po]
 ERROR [ko.po] mismatched patterns: refs/remotes/<...>/HEAD
 ERROR [ko.po] >> msgid: set refs/remotes/<name>/HEAD according to remote
 ERROR [ko.po] >> msgstr: 레퍼런스/리모트/<이름>/HEAD 값을 리모트에 맞게 설정합니다
 ERROR [ko.po]
❌ Incomplete translations found
 ERROR [ko.po] 2242 new string(s) in 'po/git.pot', but not in your 'po/XX.po'
 ERROR [ko.po]
 ERROR [ko.po] > po/git.pot:24: this message is used but not defined in po/ko.po
 ERROR [ko.po] > po/git.pot:54: this message is used but not defined in po/ko.po
 ERROR [ko.po] > po/git.pot:84: this message is used but not defined in po/ko.po
 ERROR [ko.po] > ...
 ERROR [ko.po]
 ERROR [ko.po] 568 obsolete string(s) in your 'po/XX.po', which must be removed
 ERROR [ko.po]
 ERROR [ko.po] > po/XX.po:147: warning: this message is not used
 ERROR [ko.po] > po/XX.po:172: warning: this message is not used
 ERROR [ko.po] > po/XX.po:176: warning: this message is not used
 ERROR [ko.po] > ...
 ERROR [ko.po]
 ERROR [ko.po] Please run "git-po-helper update po/XX.po" to update your po file,
 ERROR [ko.po] and translate the new strings in it.
 ERROR [ko.po]
ERROR: check-po command failed
EOF

test_expect_success "ko.po: has untranslated strings" '
	test_must_fail git -C workdir $HELPER check-po --pot-file=po/git.pot \
		--report-file-locations=none po/ko.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
