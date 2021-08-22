#!/bin/sh

test_description="check output for --github-action"

. ./lib/sharness.sh

HELPER="git-po-helper --github-action"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	test -f workdir/po/git.pot
'

cat >expect <<-\EOF
ERROR [po/zh_CN.po]    po/zh_CN.po:25: end-of-line within string
ERROR [po/zh_CN.po]    msgfmt: found 1 fatal error
ERROR [po/zh_CN.po]    fail to check po: exit status 1
ERROR [po/zh_CN.po]    fail to compile po/zh_CN.po: exit status 1
ERROR [po/zh_CN.po]    no mofile generated, and no scan typos

ERROR: fail to execute "git-po-helper check-po"
EOF

test_expect_success "bad syntax of zh_CN.po" '
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
		msgstr "po-helper 测试：不是一个真正的本地化字符串: xyz""
		EOF

		test_must_fail $HELPER check-po  zh_CN
	) >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
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

		$HELPER update zh_CN
	)
'

cat >expect <<-\EOF
INFO [po/zh_CN.po]    2 translated messages, 5102 untranslated messages.
EOF

test_expect_success "check update of zh_CN.po" '
	(
		cd workdir &&
		$HELPER check-po zh_CN
	) >out 2>&1 &&
	make_user_friendly_and_stable_output <out |
		head -2 >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
INFO [po/zh_CN.po]    2 translated messages, 5102 untranslated messages.
INFO Creating core pot file in po-core/core.pot
INFO [po-core/zh_CN.po]    2 translated messages, 479 untranslated messages.
EOF

test_expect_success "check core update of zh_CN.po" '
	(
		cd workdir &&

		$HELPER check-po --core zh_CN
	) >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
