#!/bin/sh

test_description="test git-po-helper update"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions"
POT_FILE="--pot-file=po/git.pot"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	(
		cd workdir &&
		git checkout po-2.31.1 &&
		test -f po/git.pot
	)
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=error msg="[po/zh_CN.po]    po/zh_CN.po:25: end-of-line within string"
level=error msg="[po/zh_CN.po]    msgfmt: found 1 fatal error"
level=error msg="[po/zh_CN.po]    fail to check po: exit status 1"
------------------------------------------------------------------------------
level=error msg="[po/zh_CN.po]    fail to compile po/zh_CN.po: exit status 1"
level=error msg="[po/zh_CN.po]    fail to generate mofile"
ERROR: check-po command failed
EOF

test_expect_success "bad syntax of zh_CN.po" '
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

	#: remote.c:399
	msgid "more than one receivepack given, using the first"
	msgstr "提供了一个以上的 receivepack，使用第一个"

	#: remote.c:407
	msgid "more than one uploadpack given, using the first"
	msgstr "提供了一个以上的 uploadpack，使用第一个"

	msgid "po-helper test: not a real l10n message: xyz"
	msgstr "po-helper 测试：不是一个真正的本地化字符串: xyz""
	EOF

	test_must_fail git -C workdir $HELPER check-po $POT_FILE --report-file-locations=none zh_CN >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	test_cmp expect actual
'

test_expect_success "update zh_CN successfully" '
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

	#: remote.c:399
	msgid "more than one receivepack given, using the first"
	msgstr "提供了一个以上的 receivepack，使用第一个"

	#: remote.c:407
	msgid "more than one uploadpack given, using the first"
	msgstr "提供了一个以上的 uploadpack，使用第一个"

	msgid "po-helper test: not a real l10n message: xyz"
	msgstr "po-helper 测试：不是一个真正的本地化字符串: xyz"
	EOF

	git -C workdir $HELPER update $POT_FILE zh_CN
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=info msg="[po/zh_CN.po]    2 translated messages, 5102 untranslated messages."
------------------------------------------------------------------------------
EOF

test_expect_success "check update of zh_CN.po" '
	git -C workdir $HELPER check-po $POT_FILE --report-file-locations=none zh_CN >out 2>&1 &&
	make_user_friendly_and_stable_output <out |
		head -3 >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=info msg="[po/zh_CN.po]    2 translated messages, 5102 untranslated messages."
------------------------------------------------------------------------------
level=info msg="[zh_CN.po (core)]    2 translated messages, 479 untranslated messages."
------------------------------------------------------------------------------
level=warning msg="[po/zh_CN.po]    5102 untranslated string(s) in your 'po/XX.po'"
level=warning msg="[po/zh_CN.po]"
level=warning msg="[po/zh_CN.po]     > po/XX.po:18: this message is untranslated"
level=warning msg="[po/zh_CN.po]     > po/XX.po:22: this message is untranslated"
level=warning msg="[po/zh_CN.po]     > po/XX.po:26: this message is untranslated"
level=warning msg="[po/zh_CN.po]     > ..."
level=warning msg="[po/zh_CN.po]"
EOF

test_expect_success "check core update of zh_CN.po" '
	git -C workdir $HELPER check-po $POT_FILE --report-file-locations=none --core zh_CN >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
