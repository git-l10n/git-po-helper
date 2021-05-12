#!/bin/sh

test_description="test git-po-helper update"

. ./lib/sharness.sh

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	test -f workdir/po/git.pot
'

test_expect_success "update: zh_CN.po not exist" '
	(
		cd workdir &&
		test ! -f po/zh_CN.po &&
		test_must_fail git-po-helper update zh_CN >actual 2>&1 &&
		cat >expect <<-\EOF &&
		level=error msg="fail to update \"po/zh_CN.po\", does not exist"

		ERROR: fail to execute "git-po-helper update"
		EOF
		test_cmp expect actual
	)
'

test_expect_success "fail to update zh_CN: bad syntax of zh_CN.po" '
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

		test_must_fail git-po-helper update zh_CN >out 2>&1 &&
		grep "po/zh_CN.po:25: end-of-line within string" out >actual &&
		grep "^level=error" out >>actual &&
		cat >expect <<-\EOF &&
		po/zh_CN.po:25: end-of-line within string
		level=error msg="fail to update \"po/zh_CN.po\": exit status 1"
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
		msgid "more than one receivepack given, using the first"
		msgstr "提供了一个以上的 receivepack，使用第一个"
		--
		msgid "more than one uploadpack given, using the first"
		msgstr "提供了一个以上的 uploadpack，使用第一个"
		EOF
		grep -A1 "more than one .* given, using the first" po/zh_CN.po >actual &&
		test_cmp expect actual &&

		# Mark as obsolete
		cat >expect <<-\EOF
		#~ msgid "po-helper test: not a real l10n message: xyz"
		#~ msgstr "po-helper 测试：不是一个真正的本地化字符串: xyz"
		EOF
		grep -A1 "po-helper test: " po/zh_CN.po >actual &&
		test_cmp expect actual
	)
'

test_done
