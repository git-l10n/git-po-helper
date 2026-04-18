#!/bin/sh

test_description="check typos in po files"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn --report-file-locations=none"
POT_NO="--pot-file=no"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	test -f workdir/po/git.pot
'

test_expect_success "mismatched shell variables" '
	cat >workdir/po/zh_CN.po <<-\EOF &&
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

	git -C workdir $HELPER check-po $POT_NO po/zh_CN.po >out 2>&1 &&

	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-\EOF &&
	ℹ️ Syntax check with msgfmt
	 INFO [zh_CN.po] 2 translated messages.
	ℹ️ PO filter (.gitattributes)
	 INFO [zh_CN.po] No Git `filter` attribute is set for *.po files on this path.
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] The filter attribute describes how Git should normalize #: location comments on each
	 INFO [zh_CN.po] PO entry when you commit. Those comments change often as source files move; committing
	 INFO [zh_CN.po] their churn produces noisy diffs and inflates the repository.
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] Setting filter=gettext-no-location or filter=gettext-no-line-number in .gitattributes
	 INFO [zh_CN.po] tells git-po-helper which location style you intend, so it can flag bad #: lines in
	 INFO [zh_CN.po] the PO (for example references that still include line numbers).
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] Please configure the filter for XX.po, for example:
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] .gitattributes: *.po filter=gettext-no-location
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] See:
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] https://lore.kernel.org/git/20220504124121.12683-1-worldhello.net@gmail.com/
	⚠️ msgid/msgstr pattern check
	 WARNING [zh_CN.po] mismatched patterns: $command, $res
	 WARNING [zh_CN.po] >> msgid: exit code $res from $command is < 0 or >= 128
	 WARNING [zh_CN.po] >> msgstr: 命令的退出码res 应该 < 0 或 >= 128
	 WARNING [zh_CN.po]
	 WARNING [zh_CN.po] mismatched patterns: $branch, $remote_name, $sm_path, sm_path
	 WARNING [zh_CN.po] >> msgid: Unable to find current ${remote_name}/${branch} revision in submodule path ${sm_path}
	 WARNING [zh_CN.po] >> msgstr: 无法在子模块路径 sm_path 中找到当前的 远程/分支 版本
	 WARNING [zh_CN.po]
	EOF

	test_cmp expect actual
'

test_expect_success "trash variables in msgStr (--typos=error)" '
	cat >workdir/po/zh_CN.po <<-\EOF &&
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

	msgid "exit code %d from %s is < 0 or >= 128"
	msgstr "命令 $command 的退出码 $res 应该 < 0 或 >= 128"

	EOF

	test_must_fail git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-\EOF &&
	ℹ️ Syntax check with msgfmt
	 INFO [zh_CN.po] 1 translated message.
	ℹ️ PO filter (.gitattributes)
	 INFO [zh_CN.po] No Git `filter` attribute is set for *.po files on this path.
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] The filter attribute describes how Git should normalize #: location comments on each
	 INFO [zh_CN.po] PO entry when you commit. Those comments change often as source files move; committing
	 INFO [zh_CN.po] their churn produces noisy diffs and inflates the repository.
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] Setting filter=gettext-no-location or filter=gettext-no-line-number in .gitattributes
	 INFO [zh_CN.po] tells git-po-helper which location style you intend, so it can flag bad #: lines in
	 INFO [zh_CN.po] the PO (for example references that still include line numbers).
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] Please configure the filter for XX.po, for example:
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] .gitattributes: *.po filter=gettext-no-location
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] See:
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] https://lore.kernel.org/git/20220504124121.12683-1-worldhello.net@gmail.com/
	❌ msgid/msgstr pattern check
	 ERROR [zh_CN.po] mismatched patterns: $command, $res
	 ERROR [zh_CN.po] >> msgid: exit code %d from %s is < 0 or >= 128
	 ERROR [zh_CN.po] >> msgstr: 命令 $command 的退出码 $res 应该 < 0 或 >= 128
	 ERROR [zh_CN.po]
	ERROR: check-po command failed
	EOF

	test_cmp expect actual
'

test_expect_success "check typos of mismatched constant strings" '
	cat >workdir/po/zh_CN.po <<-\EOF &&
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

	msgid "ignore invalid color %.*s in log.graphColors"
	msgstr "忽略 log.graphColorss 中无效的颜色 %.*s"

	msgid "invalid color %s in color.blame.repeatedLines"
	msgstr "color.blame.repeatedlines 中无效的颜色值 %s"

	msgid "check settings of config_variable"
	msgstr "检查配置变量的设置"

	msgid "CHERRY_PICK_HEAD exists"
	msgstr "已存在 CHERRY_PICK_HEADS"

	msgid "check settings of <config_variable>"
	msgstr "检查 <配置变量> 的设置"

	msgid "check settings of [config_variable]"
	msgstr "检查 [配置变量] 的设置"

	msgid "checking config.variables (one command)"
	msgid_plural "checking config.variables (%d commands)"
	msgstr[0] "检查 配置.变量（一条命令）"
	msgstr[1] "检查 配置.变量（%d 条命令）"

	msgid "git rebase--interactive [options]"
	msgstr "git rebase --interactive [参数]"

	msgid "git-credential--helper [options]"
	msgstr "git-credential-helper [参数]"
	EOF

	git -C workdir $HELPER check-po $POT_NO po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-\EOF &&
	ℹ️ Syntax check with msgfmt
	 INFO [zh_CN.po] 9 translated messages.
	ℹ️ PO filter (.gitattributes)
	 INFO [zh_CN.po] No Git `filter` attribute is set for *.po files on this path.
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] The filter attribute describes how Git should normalize #: location comments on each
	 INFO [zh_CN.po] PO entry when you commit. Those comments change often as source files move; committing
	 INFO [zh_CN.po] their churn produces noisy diffs and inflates the repository.
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] Setting filter=gettext-no-location or filter=gettext-no-line-number in .gitattributes
	 INFO [zh_CN.po] tells git-po-helper which location style you intend, so it can flag bad #: lines in
	 INFO [zh_CN.po] the PO (for example references that still include line numbers).
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] Please configure the filter for XX.po, for example:
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] .gitattributes: *.po filter=gettext-no-location
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] See:
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] https://lore.kernel.org/git/20220504124121.12683-1-worldhello.net@gmail.com/
	⚠️ msgid/msgstr pattern check
	 WARNING [zh_CN.po] mismatched patterns: log.graphColors, log.graphColorss
	 WARNING [zh_CN.po] >> msgid: ignore invalid color %.*s in log.graphColors
	 WARNING [zh_CN.po] >> msgstr: 忽略 log.graphColorss 中无效的颜色 %.*s
	 WARNING [zh_CN.po]
	 WARNING [zh_CN.po] mismatched patterns: color.blame.repeatedLines, color.blame.repeatedlines
	 WARNING [zh_CN.po] >> msgid: invalid color %s in color.blame.repeatedLines
	 WARNING [zh_CN.po] >> msgstr: color.blame.repeatedlines 中无效的颜色值 %s
	 WARNING [zh_CN.po]
	 WARNING [zh_CN.po] mismatched patterns: config_variable
	 WARNING [zh_CN.po] >> msgid: check settings of config_variable
	 WARNING [zh_CN.po] >> msgstr: 检查配置变量的设置
	 WARNING [zh_CN.po]
	 WARNING [zh_CN.po] mismatched patterns: CHERRY_PICK_HEAD, CHERRY_PICK_HEADS
	 WARNING [zh_CN.po] >> msgid: CHERRY_PICK_HEAD exists
	 WARNING [zh_CN.po] >> msgstr: 已存在 CHERRY_PICK_HEADS
	 WARNING [zh_CN.po]
	 WARNING [zh_CN.po] mismatched patterns: config.variables
	 WARNING [zh_CN.po] >> msgid: checking config.variables (one command)
	 WARNING [zh_CN.po] >> msgstr: 检查 配置.变量（一条命令）
	 WARNING [zh_CN.po]
	 WARNING [zh_CN.po] mismatched patterns: config.variables
	 WARNING [zh_CN.po] >> msgid: checking config.variables (%d commands)
	 WARNING [zh_CN.po] >> msgstr: 检查 配置.变量（%d 条命令）
	 WARNING [zh_CN.po]
	 WARNING [zh_CN.po] mismatched patterns: --interactive, git rebase--interactive
	 WARNING [zh_CN.po] >> msgid: git rebase--interactive [options]
	 WARNING [zh_CN.po] >> msgstr: git rebase --interactive [参数]
	 WARNING [zh_CN.po]
	 WARNING [zh_CN.po] mismatched patterns: git-credential--helper, git-credential-helper
	 WARNING [zh_CN.po] >> msgid: git-credential--helper [options]
	 WARNING [zh_CN.po] >> msgstr: git-credential-helper [参数]
	 WARNING [zh_CN.po]
	EOF
	test_cmp expect actual
'

test_expect_success "check typos of mismatched options" '
	cat >workdir/po/zh_CN.po <<-\EOF &&
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

	msgid "--reject and --3way cannot be used together."
	msgstr "--reject 和 -3way 不能同时使用。"

	msgid "mark new files with `git add --intent-to-add`"
	msgstr "使用命令 `git add --intent-to-addd` 标记新增文件"

	msgid "equivalent to --word-diff=color --word-diff-regex=<regex>"
	msgstr "相当于 --word-diff=color --word-diff-regex=正则"
	EOF

	git -C workdir $HELPER check-po $POT_NO po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-\EOF &&
	ℹ️ Syntax check with msgfmt
	 INFO [zh_CN.po] 3 translated messages.
	ℹ️ PO filter (.gitattributes)
	 INFO [zh_CN.po] No Git `filter` attribute is set for *.po files on this path.
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] The filter attribute describes how Git should normalize #: location comments on each
	 INFO [zh_CN.po] PO entry when you commit. Those comments change often as source files move; committing
	 INFO [zh_CN.po] their churn produces noisy diffs and inflates the repository.
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] Setting filter=gettext-no-location or filter=gettext-no-line-number in .gitattributes
	 INFO [zh_CN.po] tells git-po-helper which location style you intend, so it can flag bad #: lines in
	 INFO [zh_CN.po] the PO (for example references that still include line numbers).
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] Please configure the filter for XX.po, for example:
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] .gitattributes: *.po filter=gettext-no-location
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] See:
	 INFO [zh_CN.po]
	 INFO [zh_CN.po] https://lore.kernel.org/git/20220504124121.12683-1-worldhello.net@gmail.com/
	⚠️ msgid/msgstr pattern check
	 WARNING [zh_CN.po] mismatched patterns: --3way
	 WARNING [zh_CN.po] >> msgid: --reject and --3way cannot be used together.
	 WARNING [zh_CN.po] >> msgstr: --reject 和 -3way 不能同时使用。
	 WARNING [zh_CN.po]
	 WARNING [zh_CN.po] mismatched patterns: --intent-to-add, --intent-to-addd
	 WARNING [zh_CN.po] >> msgid: mark new files with `git add --intent-to-add`
	 WARNING [zh_CN.po] >> msgstr: 使用命令 `git add --intent-to-addd` 标记新增文件
	 WARNING [zh_CN.po]
	 WARNING [zh_CN.po] mismatched patterns: --word-diff-regex, --word-diff-regex=<...>
	 WARNING [zh_CN.po] >> msgid: equivalent to --word-diff=color --word-diff-regex=<regex>
	 WARNING [zh_CN.po] >> msgstr: 相当于 --word-diff=color --word-diff-regex=正则
	 WARNING [zh_CN.po]
	EOF

	test_cmp expect actual
'

test_done
