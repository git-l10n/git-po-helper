#!/bin/sh

test_description="test git-po-helper check-commits with typos"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn"
POT_NO="--pot-file=no"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	(
		cd workdir &&
		git checkout po-2.31.1 &&
		test_tick &&
		git tag -m v1 v1 &&
		test -f po/git.pot
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
		git tag -m v2 v2
	)
'

test_expect_success "check-commits show typos" '
	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO v1.. >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-\EOF &&
	ℹ️ Syntax check with msgfmt
	 INFO [zh_CN.po@rev] 2 translated messages.
	❌ PO filter (.gitattributes)
	 ERROR [zh_CN.po@rev] No filter attribute set for XX.po. This will introduce location newlines into the
	 ERROR [zh_CN.po@rev] repository and cause repository bloat.
	 ERROR [zh_CN.po@rev]
	 ERROR [zh_CN.po@rev] Please configure the filter attribute for XX.po, for example:
	 ERROR [zh_CN.po@rev]
	 ERROR [zh_CN.po@rev] .gitattributes: *.po filter=gettext-no-location
	 ERROR [zh_CN.po@rev]
	 ERROR [zh_CN.po@rev] See:
	 ERROR [zh_CN.po@rev]
	 ERROR [zh_CN.po@rev] https://lore.kernel.org/git/20220504124121.12683-1-worldhello.net@gmail.com/
	⚠️ msgid/msgstr pattern check
	 WARNING [zh_CN.po@rev] mismatched patterns: $command, $res
	 WARNING [zh_CN.po@rev] >> msgid: exit code $res from $command is < 0 or >= 128
	 WARNING [zh_CN.po@rev] >> msgstr: 命令的退出码res 应该 < 0 或 >= 128
	 WARNING [zh_CN.po@rev]
	 WARNING [zh_CN.po@rev] mismatched patterns: $branch, $remote_name, $sm_path, sm_path
	 WARNING [zh_CN.po@rev] >> msgid: Unable to find current ${remote_name}/${branch} revision in submodule path ${sm_path}
	 WARNING [zh_CN.po@rev] >> msgstr: 无法在子模块路径 sm_path 中找到当前的 远程/分支 版本
	 WARNING [zh_CN.po@rev]
	⚠️ Author and committer
	 WARNING commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different
	INFO: checking commits: 0 passed, 1 failed.
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "check-commits show typos (--typos=error)" '
	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO --report-typos=error v1.. >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-\EOF &&
	ℹ️ Syntax check with msgfmt
	 INFO [zh_CN.po@rev] 2 translated messages.
	❌ PO filter (.gitattributes)
	 ERROR [zh_CN.po@rev] No filter attribute set for XX.po. This will introduce location newlines into the
	 ERROR [zh_CN.po@rev] repository and cause repository bloat.
	 ERROR [zh_CN.po@rev]
	 ERROR [zh_CN.po@rev] Please configure the filter attribute for XX.po, for example:
	 ERROR [zh_CN.po@rev]
	 ERROR [zh_CN.po@rev] .gitattributes: *.po filter=gettext-no-location
	 ERROR [zh_CN.po@rev]
	 ERROR [zh_CN.po@rev] See:
	 ERROR [zh_CN.po@rev]
	 ERROR [zh_CN.po@rev] https://lore.kernel.org/git/20220504124121.12683-1-worldhello.net@gmail.com/
	❌ msgid/msgstr pattern check
	 ERROR [zh_CN.po@rev] mismatched patterns: $command, $res
	 ERROR [zh_CN.po@rev] >> msgid: exit code $res from $command is < 0 or >= 128
	 ERROR [zh_CN.po@rev] >> msgstr: 命令的退出码res 应该 < 0 或 >= 128
	 ERROR [zh_CN.po@rev]
	 ERROR [zh_CN.po@rev] mismatched patterns: $branch, $remote_name, $sm_path, sm_path
	 ERROR [zh_CN.po@rev] >> msgid: Unable to find current ${remote_name}/${branch} revision in submodule path ${sm_path}
	 ERROR [zh_CN.po@rev] >> msgstr: 无法在子模块路径 sm_path 中找到当前的 远程/分支 版本
	 ERROR [zh_CN.po@rev]
	⚠️ Author and committer
	 WARNING commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different
	INFO: checking commits: 0 passed, 1 failed.
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "update po/TEAMS" '
	(
		cd workdir &&
		echo >>po/TEAMS &&
		git add -u &&
		test_tick &&
		git commit -s -m "l10n: TEAMS: update for test"
	)
'

test_expect_success "check-commits show typos and TEAMS file" '
	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO v1.. >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-\EOF &&
	❌ Changes outside po/
	 ERROR commit <OID>: bad syntax at po/TEAMS:79 (unknown key "Respository"): Respository:    https://github.com/l10n-tw/git-po
	 ERROR commit <OID>: bad syntax at po/TEAMS:80 (need two tabs between k/v): Leader:     Yi-Jyun Pan <pan93412 AT gmail.com>
	⚠️ Author and committer
	 WARNING commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different
	ℹ️ Syntax check with msgfmt
	 INFO [zh_CN.po@rev] 2 translated messages.
	❌ PO filter (.gitattributes)
	 ERROR [zh_CN.po@rev] No filter attribute set for XX.po. This will introduce location newlines into the
	 ERROR [zh_CN.po@rev] repository and cause repository bloat.
	 ERROR [zh_CN.po@rev]
	 ERROR [zh_CN.po@rev] Please configure the filter attribute for XX.po, for example:
	 ERROR [zh_CN.po@rev]
	 ERROR [zh_CN.po@rev] .gitattributes: *.po filter=gettext-no-location
	 ERROR [zh_CN.po@rev]
	 ERROR [zh_CN.po@rev] See:
	 ERROR [zh_CN.po@rev]
	 ERROR [zh_CN.po@rev] https://lore.kernel.org/git/20220504124121.12683-1-worldhello.net@gmail.com/
	⚠️ msgid/msgstr pattern check
	 WARNING [zh_CN.po@rev] mismatched patterns: $command, $res
	 WARNING [zh_CN.po@rev] >> msgid: exit code $res from $command is < 0 or >= 128
	 WARNING [zh_CN.po@rev] >> msgstr: 命令的退出码res 应该 < 0 或 >= 128
	 WARNING [zh_CN.po@rev]
	 WARNING [zh_CN.po@rev] mismatched patterns: $branch, $remote_name, $sm_path, sm_path
	 WARNING [zh_CN.po@rev] >> msgid: Unable to find current ${remote_name}/${branch} revision in submodule path ${sm_path}
	 WARNING [zh_CN.po@rev] >> msgstr: 无法在子模块路径 sm_path 中找到当前的 远程/分支 版本
	 WARNING [zh_CN.po@rev]
	⚠️ Author and committer
	 WARNING commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different
	INFO: checking commits: 0 passed, 2 failed.
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_done
