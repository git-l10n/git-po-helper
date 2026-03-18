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

	test_must_fail git -C workdir $HELPER check-po $POT_FILE --report-file-locations=none po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	❌ Syntax check with msgfmt
	 ERROR [zh_CN.po] po/zh_CN.po:25: end-of-line within string
	 ERROR [zh_CN.po] msgfmt: found 1 fatal error
	 ERROR [zh_CN.po] fail to check po: exit status 1
	❌ Incomplete translations found
	 ERROR [zh_CN.po] 5102 new string(s) in ${SQ}po/git.pot${SQ}, but not in your ${SQ}po/XX.po${SQ}
	 ERROR [zh_CN.po]
	 ERROR [zh_CN.po] > po/git.pot: %-*s forces to %-*s (%s)
	 ERROR [zh_CN.po] > po/git.pot: %-*s forces to %s
	 ERROR [zh_CN.po] > po/git.pot: %-*s pushes to %-*s (%s)
	 ERROR [zh_CN.po] > ...
	 ERROR [zh_CN.po]
	 ERROR [zh_CN.po] 1 obsolete string(s) in your ${SQ}po/XX.po${SQ}, which must be removed
	 ERROR [zh_CN.po]
	 ERROR [zh_CN.po] > po/XX.po:po-helper test: not a real l10...
	 ERROR [zh_CN.po]
	 ERROR [zh_CN.po] Please run "git-po-helper update po/XX.po" to update your po file,
	 ERROR [zh_CN.po] and translate the new strings in it.
	 ERROR [zh_CN.po]
	ERROR: check-po command failed
	EOF

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
	EOF

	git -C workdir $HELPER update $POT_FILE zh_CN
'

test_expect_success "check update of zh_CN.po" '
	git -C workdir $HELPER check-po $POT_FILE --report-file-locations=none po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out |
		head -3 >actual &&

	cat >expect <<-\EOF &&
	ℹ️ Syntax check with msgfmt
	 INFO [zh_CN.po] 2 translated messages, 5102 untranslated messages.
	⚠️ Incomplete translations found
	EOF

	test_cmp expect actual
'

test_expect_success "check core update of zh_CN.po" '
	git -C workdir $HELPER check-po $POT_FILE --report-file-locations=none --core po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	ℹ️ Syntax check with msgfmt
	 INFO [zh_CN.po] 2 translated messages, 5102 untranslated messages.
	⚠️ Incomplete translations found
	 WARNING [zh_CN.po] 5102 untranslated string(s) in your ${SQ}po/XX.po${SQ}
	 WARNING [zh_CN.po]
	 WARNING [zh_CN.po] > po/XX.po:Huh (%s)?
	 WARNING [zh_CN.po] > po/XX.po:could not read index
	 WARNING [zh_CN.po] > po/XX.po:binary
	 WARNING [zh_CN.po] > ...
	 WARNING [zh_CN.po]
	ℹ️ Core PO vs git-core.pot
	 INFO [zh_CN.po (core)] 2 translated messages, 479 untranslated messages.
	EOF

	test_cmp expect actual
'

test_done
