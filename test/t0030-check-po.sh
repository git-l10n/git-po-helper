#!/bin/sh

test_description="test git-po-helper update"

. ./lib/sharness.sh

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	test -f workdir/po/git.pot
'

test_expect_success "bad syntax of zh_CN.po" '
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

		#: remote.c:399
		msgid "more than one receivepack given, using the first"
		msgstr "提供了一个以上的 receivepack，使用第一个"

		#: remote.c:407
		msgid "more than one uploadpack given, using the first"
		msgstr "提供了一个以上的 uploadpack，使用第一个"

		msgid "po-helper test: not a real l10n message: xyz"
		msgstr "po-helper 测试：不是一个真正的本地化字符串: xyz""
		EOF

		test_must_fail git-po-helper check-po  zh_CN >out 2>&1 &&
		make_user_friendly_and_stable_output <out >actual &&
		cat >expect <<-\EOF &&
		level=info msg="Checking syntax of po file for \"Chinese - China\""
		level=error msg="Fail to check \"po/zh_CN.po\": exit status 1"
		level=error msg="    po/zh_CN.po:25: end-of-line within string\n"
		level=error msg="    msgfmt: found 1 fatal error\n"

		ERROR: fail to execute "git-po-helper check-po"
		EOF
		test_cmp expect actual
	)
'

test_expect_success "update zh_CN successfully" '
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

		#: remote.c:399
		msgid "more than one receivepack given, using the first"
		msgstr "提供了一个以上的 receivepack，使用第一个"

		#: remote.c:407
		msgid "more than one uploadpack given, using the first"
		msgstr "提供了一个以上的 uploadpack，使用第一个"

		msgid "po-helper test: not a real l10n message: xyz"
		msgstr "po-helper 测试：不是一个真正的本地化字符串: xyz"
		EOF

		git-po-helper update zh_CN
	)
'

test_expect_success "check update of zh_CN.po" '
	(
		cd workdir &&

		cat >expect <<-\EOF
		[po/zh_CN.po] 2 translated messages, 5102 untranslated messages.
		EOF
		git-po-helper check-po zh_CN >out 2>&1 &&
		make_user_friendly_and_stable_output <out |
			head -2 >actual &&
		test_cmp expect actual
	)
'

test_expect_success "check core update of zh_CN.po" '
	(
		cd workdir &&

		cat >expect <<-\EOF
		level=info msg="Creating core pot file in po-core/core.pot"
		[po-core/zh_CN.po] 2 translated messages, 479 untranslated messages.
		EOF
		git-po-helper check-po --core zh_CN >out 2>&1 &&
		grep -A1 "Creating core pot file" out >actual &&
		test_cmp expect actual
	)
'

test_expect_success "show warning of gettext 0.14 not found issue" '
	(
		cd workdir &&

		cat >expect <<-\EOF
		level=warning msg="cannot find gettext 0.14, and won'"'"'t run gettext backward compatible test"
		[po/zh_CN.po] 2 translated messages, 5102 untranslated messages.
		EOF
		git-po-helper check-po --no-gettext-back-compatible zh_CN >actual 2>&1 &&
		test_cmp expect actual
	)
'

test_done
