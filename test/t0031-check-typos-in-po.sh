#!/bin/sh

test_description="check typos in po files"

. ./lib/sharness.sh

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	test -f workdir/po/git.pot
'

test_expect_success "check typos in zh_CN.po" '
	(
		cd workdir &&
		test ! -f po/zh_CN.po &&

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

		msgid "check settings of core.gitProxy config variable"
		msgstr "检查 core.gitProxy 配置变量的设置"

		msgid "check settings of config_variable"
		msgstr "检查配置变量的设置"

		msgid "check settings of <config_variable>"
		msgstr "检查 <配置变量> 的设置"

		msgid "check settings of [config_variable]"
		msgstr "检查 [配置变量] 的设置"

		msgid "checking config.variables (one command)"
		msgid_plural "checking config.variables (%d commands)"
		msgstr[0] "检查 配置.变量（一条命令）"
		msgstr[1] "检查 配置.变量（%d 条命令）"
		EOF

		git-po-helper check-po  zh_CN >out 2>&1 &&
		make_user_friendly_and_stable_output <out >actual &&
		cat >expect <<-\EOF &&
		[po/zh_CN.po] 6 translated messages.
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
		level=warning msg="mismatch variable names: log.graphColors"
		level=warning msg=">> msgid: ignore invalid color %.*s in log.graphColors"
		level=warning msg=">> msgstr: 忽略 log.graphColorss 中无效的颜色 %.*s"
		level=warning
		EOF
		test_cmp expect actual
	)
'

test_done
