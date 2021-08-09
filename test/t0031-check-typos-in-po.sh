#!/bin/sh

test_description="check typos in po files"

. ./lib/sharness.sh

HELPER="git-po-helper --no-gettext-back-compatible"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	test -f workdir/po/git.pot
'

test_expect_success "mismatched shell variables" '
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

		$HELPER check-po  zh_CN >out 2>&1 &&
		make_user_friendly_and_stable_output <out >actual &&
		cat >expect <<-\EOF &&
		[po/zh_CN.po] 2 translated messages.
		level=warning msg="mismatch variable names: ${remote_name}, ${branch}, ${sm_path}"
		level=warning msg=">> msgid: Unable to find current ${remote_name}/${branch} revision in submodule path ${sm_path}"
		level=warning msg=">> msgstr: 无法在子模块路径 sm_path 中找到当前的 远程/分支 版本"
		level=warning
		level=warning msg="mismatch variable names: $res, $command"
		level=warning msg=">> msgid: exit code $res from $command is < 0 or >= 128"
		level=warning msg=">> msgstr: 命令的退出码res 应该 < 0 或 >= 128"
		level=warning
		EOF
		test_cmp expect actual
	)
'

test_expect_success "check typos of mismatched constant strings" '
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

		$HELPER check-po  zh_CN >out 2>&1 &&
		make_user_friendly_and_stable_output <out >actual &&
		cat >expect <<-\EOF &&
		[po/zh_CN.po] 9 translated messages.
		level=warning msg="mismatch variable names: CHERRY_PICK_HEAD"
		level=warning msg=">> msgid: CHERRY_PICK_HEAD exists"
		level=warning msg=">> msgstr: 已存在 CHERRY_PICK_HEADS"
		level=warning
		level=warning msg="mismatch variable names: config_variable"
		level=warning msg=">> msgid: check settings of config_variable"
		level=warning msg=">> msgstr: 检查配置变量的设置"
		level=warning
		level=warning msg="mismatch variable names: config.variables"
		level=warning msg=">> msgid: checking config.variables (one command)"
		level=warning msg=">> msgstr: 检查 配置.变量（一条命令）"
		level=warning
		level=warning msg="mismatch variable names: config.variables"
		level=warning msg=">> msgid: checking config.variables (%d commands)"
		level=warning msg=">> msgstr: 检查 配置.变量（%d 条命令）"
		level=warning
		level=warning msg="mismatch variable names: rebase--interactive"
		level=warning msg=">> msgid: git rebase--interactive [options]"
		level=warning msg=">> msgstr: git rebase --interactive [参数]"
		level=warning
		level=warning msg="mismatch variable names: git-credential--helper"
		level=warning msg=">> msgid: git-credential--helper [options]"
		level=warning msg=">> msgstr: git-credential-helper [参数]"
		level=warning
		level=warning msg="mismatch variable names: log.graphColors"
		level=warning msg=">> msgid: ignore invalid color %.*s in log.graphColors"
		level=warning msg=">> msgstr: 忽略 log.graphColorss 中无效的颜色 %.*s"
		level=warning
		level=warning msg="mismatch variable names: color.blame.repeatedLines"
		level=warning msg=">> msgid: invalid color %s in color.blame.repeatedLines"
		level=warning msg=">> msgstr: color.blame.repeatedlines 中无效的颜色值 %s"
		level=warning
		EOF
		test_cmp expect actual
	)
'

test_done
