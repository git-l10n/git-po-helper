#!/bin/sh

test_description="test git-po-helper check-commits in bare repo"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn"
POT_NO="--pot-file=no"

test_expect_success "setup" '
	git clone --mirror "$PO_HELPER_TEST_REPOSITORY" repo.git &&
	git clone repo.git workdir &&
	(
		cd workdir &&
		git switch po-2.31.1 &&
		test_tick &&
		git tag -m v1 v1 &&
		git push origin --tags
	)
'

test_expect_success "create po/zh_CN with typos" '
	(
		cd workdir &&
		cat >po/zh_CN.po <<-\EOF &&
		msgid ""
		msgstr ""
		"Project-Id-Version: Git\n"
		"Report-Msgid-Bugs-To: Git Mailing List <git@vger.kernel.org>\n"
		"POT-Creation-Date: 2021-03-04 22:41+0800\n"
		"PO-Revision-Date: 2021-03-04 22:41+0800\n"
		"Last-Translator: Automatically generated\n"
		"Language-Team: none\n"
		"Language: zh_CN\n"
		"MIME-Version: 1.0\n"
		"Content-Type: text/plain; charset=UTF-8\n"
		"Content-Transfer-Encoding: 8bit\n"
		"Plural-Forms: nplurals=2; plural=(n != 1);\n"

		msgid "exit code $res from $command is < 0 or >= 128"
		msgstr "命令的退出码res 应该 < 0 或 >= 128"

		msgid ""
		"Unable to find current ${remote_name}/${branch} revision in submodule path "
		"${sm_path}"
		msgstr ""
		"无法在子模块路径 sm_path 中找到当前的 远程/分支 版本"
		EOF

		git add "po/zh_CN.po" &&
		test_tick &&
		git commit -s -m "l10n: add po/zh_CN" &&
		git tag -m v2 v2 &&
		git push origin --tags HEAD
	)
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=info msg="[po/zh_CN.po@rev]    2 translated messages."
------------------------------------------------------------------------------
level=warning msg="[po/zh_CN.po@rev]    mismatched patterns: $branch, $remote_name, $sm_path, sm_path"
level=warning msg="[po/zh_CN.po@rev]    >> msgid: Unable to find current ${remote_name}/${branch} revision in submodule path ${sm_path}"
level=warning msg="[po/zh_CN.po@rev]    >> msgstr: 无法在子模块路径 sm_path 中找到当前的 远程/分支 版本"
level=warning msg="[po/zh_CN.po@rev]"
level=warning msg="[po/zh_CN.po@rev]    mismatched patterns: $command, $res"
level=warning msg="[po/zh_CN.po@rev]    >> msgid: exit code $res from $command is < 0 or >= 128"
level=warning msg="[po/zh_CN.po@rev]    >> msgstr: 命令的退出码res 应该 < 0 或 >= 128"
level=warning msg="[po/zh_CN.po@rev]"
------------------------------------------------------------------------------
level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
level=info msg="checking commits: 1 passed."
EOF

test_expect_success "check-commits show typos" '
	git -C repo.git $HELPER check-commits $POT_NO v1..v2 >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=info msg="[po/zh_CN.po@rev]    2 translated messages."
------------------------------------------------------------------------------
level=error msg="[po/zh_CN.po@rev]    mismatched patterns: $branch, $remote_name, $sm_path, sm_path"
level=error msg="[po/zh_CN.po@rev]    >> msgid: Unable to find current ${remote_name}/${branch} revision in submodule path ${sm_path}"
level=error msg="[po/zh_CN.po@rev]    >> msgstr: 无法在子模块路径 sm_path 中找到当前的 远程/分支 版本"
level=error msg="[po/zh_CN.po@rev]"
level=error msg="[po/zh_CN.po@rev]    mismatched patterns: $command, $res"
level=error msg="[po/zh_CN.po@rev]    >> msgid: exit code $res from $command is < 0 or >= 128"
level=error msg="[po/zh_CN.po@rev]    >> msgstr: 命令的退出码res 应该 < 0 或 >= 128"
level=error msg="[po/zh_CN.po@rev]"
------------------------------------------------------------------------------
level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
level=info msg="checking commits: 0 passed, 1 failed."
ERROR: check-commits command failed
EOF

test_expect_success "check-commits show typos (--typos=error)" '
	test_must_fail git -C repo.git $HELPER \
		check-commits $POT_NO --report-typos=error v1..v2 >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "update po/TEAMS" '
	(
		cd workdir &&
		echo >>po/TEAMS &&
		git add -u &&
		test_tick &&
		git commit -s -m "l10n: TEAMS: update for test" &&
		git tag -m v3 v3 &&
		git push origin --tags HEAD
	)
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=error msg="commit <OID>: bad syntax at po/TEAMS:79 (unknown key \"Respository\"): Respository:    https://github.com/l10n-tw/git-po"
level=error msg="commit <OID>: bad syntax at po/TEAMS:80 (need two tabs between k/v): Leader:     Yi-Jyun Pan <pan93412 AT gmail.com>"
------------------------------------------------------------------------------
level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
------------------------------------------------------------------------------
level=info msg="[po/zh_CN.po@rev]    2 translated messages."
------------------------------------------------------------------------------
level=warning msg="[po/zh_CN.po@rev]    mismatched patterns: $branch, $remote_name, $sm_path, sm_path"
level=warning msg="[po/zh_CN.po@rev]    >> msgid: Unable to find current ${remote_name}/${branch} revision in submodule path ${sm_path}"
level=warning msg="[po/zh_CN.po@rev]    >> msgstr: 无法在子模块路径 sm_path 中找到当前的 远程/分支 版本"
level=warning msg="[po/zh_CN.po@rev]"
level=warning msg="[po/zh_CN.po@rev]    mismatched patterns: $command, $res"
level=warning msg="[po/zh_CN.po@rev]    >> msgid: exit code $res from $command is < 0 or >= 128"
level=warning msg="[po/zh_CN.po@rev]    >> msgstr: 命令的退出码res 应该 < 0 或 >= 128"
level=warning msg="[po/zh_CN.po@rev]"
------------------------------------------------------------------------------
level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
level=info msg="checking commits: 1 passed, 1 failed."
ERROR: check-commits command failed
EOF

test_expect_success "check-commits show typos and TEAMS file" '
	test_must_fail git -C repo.git $HELPER check-commits $POT_NO v1..v3 >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
