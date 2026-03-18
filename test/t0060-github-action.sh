#!/bin/sh

test_description="check output for --github-action-event"

. ./lib/test-lib.sh

HELPER="po-helper --github-action-event=pull_request_target"
POT_FILE="--pot-file=po/git.pot"
POT_NO="--pot-file=no"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	(
		cd workdir &&
		git switch po-2.31.1 &&
		test -f po/git.pot
	)
'

cat >expect <<-\EOF
❌ Syntax check with msgfmt
 ERROR [zh_CN.po] po/zh_CN.po:25: end-of-line within string
 ERROR [zh_CN.po] msgfmt: found 1 fatal error
 ERROR [zh_CN.po] fail to check po: exit status 1
❌ Location comments (#:)
 ERROR [zh_CN.po] entry 1 (msgid "more than one receivepack g..."): location comment contains line number (use file-only or remove): "remote.c:399"
❌ PO filter (.gitattributes)
 ERROR [zh_CN.po] No filter attribute set for XX.po. This will introduce location newlines into the
 ERROR [zh_CN.po] repository and cause repository bloat.
 ERROR [zh_CN.po]
 ERROR [zh_CN.po] Please configure the filter attribute for XX.po, for example:
 ERROR [zh_CN.po]
 ERROR [zh_CN.po] .gitattributes: *.po filter=gettext-no-location
 ERROR [zh_CN.po]
 ERROR [zh_CN.po] See:
 ERROR [zh_CN.po]
 ERROR [zh_CN.po] https://lore.kernel.org/git/20220504124121.12683-1-worldhello.net@gmail.com/
❌ Incomplete translations found
 ERROR [zh_CN.po] 5102 new string(s) in 'po/git.pot', but not in your 'po/XX.po'
 ERROR [zh_CN.po]
 ERROR [zh_CN.po] > po/git.pot: %-*s forces to %-*s (%s)
 ERROR [zh_CN.po] > po/git.pot: %-*s forces to %s
 ERROR [zh_CN.po] > po/git.pot: %-*s pushes to %-*s (%s)
 ERROR [zh_CN.po] > ...
 ERROR [zh_CN.po]
 ERROR [zh_CN.po] 1 obsolete string(s) in your 'po/XX.po', which must be removed
 ERROR [zh_CN.po]
 ERROR [zh_CN.po] > po/XX.po:po-helper test: not a real l10...
 ERROR [zh_CN.po]
 ERROR [zh_CN.po] Please run "git-po-helper update po/XX.po" to update your po file,
 ERROR [zh_CN.po] and translate the new strings in it.
 ERROR [zh_CN.po]
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

	test_must_fail git -C workdir $HELPER \
		check-po $POT_FILE po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	test_cmp expect actual
'

test_expect_success "update zh_CN (--add-location=file)" '
	cat >expect <<-EOF &&
	INFO run msgmerge for "Chinese - China": msgmerge --add-location=file -o - po/zh_CN.po po/git.pot
	ℹ️ Syntax check with msgfmt
	 INFO [zh_CN.po] 2 translated messages, 5102 untranslated messages.
	EOF

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

	git -C workdir $HELPER update $POT_FILE zh_CN 2>&1 |
		make_user_friendly_and_stable_output |
		sed "/^\.\./ d" >actual &&
	test_cmp expect actual
'

test_expect_success "check update of zh_CN.po" '
	cat >expect <<-\EOF &&
	ℹ️ Syntax check with msgfmt
	 INFO [zh_CN.po] 2 translated messages, 5102 untranslated messages.
	EOF

	git -C workdir $HELPER \
		check-po $POT_FILE --report-file-locations=none po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out |
		head -2 >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
ℹ️ Syntax check with msgfmt
 INFO [zh_CN.po] 2 translated messages, 5102 untranslated messages.
ℹ️ Core PO vs git-core.pot
 INFO [zh_CN.po (core)] 2 translated messages, 479 untranslated messages.
⚠️ Incomplete translations found
 WARNING [zh_CN.po] 5102 untranslated string(s) in your 'po/XX.po'
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] > po/XX.po:Huh (%s)?
 WARNING [zh_CN.po] > po/XX.po:could not read index
 WARNING [zh_CN.po] > po/XX.po:binary
 WARNING [zh_CN.po] > ...
 WARNING [zh_CN.po]
EOF

test_expect_success "check core update of zh_CN.po" '
	git -C workdir $HELPER \
		check-po $POT_FILE --core --report-file-locations=none po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
⚠️ Author and committer
 WARNING commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different
❌ Commit subject
 ERROR commit <OID>: subject ("Add files ...") does not have prefix "l10n:"
❌ Commit message body
 ERROR commit <OID>: empty body of the commit message, no s-o-b signature
INFO checking commits: 0 passed, 1 failed.
ERROR: check-commits command failed
EOF

test_expect_success "check-commits (old-oid is zero)" '
	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO 0000000000000000000000000000000000000000..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "create new non-l10n commit" '
	(
		cd workdir &&
		echo A >A.txt &&
		git add A.txt &&
		test_tick &&
		git commit -m "A"
	)
'

cat >expect <<-\EOF
❌ Changes outside po/
 ERROR commit <OID>: found changes beyond "po/" directory:
 ERROR         A.txt
 ERROR
 ERROR commit <OID>: break because this commit is not for git-l10n
INFO checking commits: 0 passed, 1 failed, 1 skipped.
ERROR: check-commits command failed
EOF

test_expect_success "check-commits (non-l10n commit)" '
	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO 0000000000000000000000000000000000000000..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "check-commits --github-action-event=pull_request" '
	test_must_fail git -C workdir po-helper \
		check-commits --pot-file=no \
		--github-action-event=pull_request \
		0000000000000000000000000000000000000000..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "check-commits --github-action-event=pull_request_target" '
	test_must_fail git -C workdir po-helper \
		check-commits --pot-file=no \
		--github-action-event=pull_request_target \
		0000000000000000000000000000000000000000..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
⚠️ Changes outside po/
 WARNING commit <OID>: found changes beyond "po/" directory:
 WARNING         A.txt
 WARNING
 WARNING commit <OID>: break because this commit is not for git-l10n
INFO checking commits: 0 passed, 0 failed, 2 skipped.
EOF

test_expect_success "check-commits --github-action-event=push" '
	git -C workdir po-helper \
		check-commits --pot-file=no \
		--github-action-event push \
		0000000000000000000000000000000000000000..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "check-commits --github-action-event=push" '
	git -C workdir po-helper \
		check-commits --pot-file=no \
		--github-action-event push \
		0000000000000000000000000000000000000000..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
