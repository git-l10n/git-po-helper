#!/bin/sh

test_description="show gettext 0.14 incompatible errors"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions"
POT_NO="--pot-file=no"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1 &&
	(
		cd workdir &&
		cat >po/fr.po <<-\EOF
		# This example is from v2.21.0:po/fr.po
		msgid ""
		msgstr ""
		"Project-Id-Version: git\n"
		"Report-Msgid-Bugs-To: Git Mailing List <git@vger.kernel.org>\n"
		"POT-Creation-Date: 2019-02-15 10:09+0800\n"
		"PO-Revision-Date: 2019-02-15 22:18+0100\n"
		"Last-Translator: Jean-Noël Avila <jn.avila@free.fr>\n"
		"Language-Team: Jean-Noël Avila <jn.avila@free.fr>\n"
		"Language: fr\n"
		"MIME-Version: 1.0\n"
		"Content-Type: text/plain; charset=UTF-8\n"
		"Content-Transfer-Encoding: 8bit\n"
		"Plural-Forms: nplurals=2; plural=n<=1 ?0 : 1;\n"

		#: advice.c:101
		#, c-format
		msgid "%shint: %.*s%s\n"
		msgstr "%sastuce: %.*s%s\n"

		#. TRANSLATORS: please keep "[y|N]" as is.
		#: git-send-email.perl:1945
		#, perl-format
		msgid "Do you really want to send %s? [y|N]: "
		msgstr "Souhaitez-vous réellement envoyer %s ?[y|N] : "

		#, fuzzy
		#~| msgid "invalid sparse value '%s'"
		#~ msgid "invalid --stat value: %s"
		#~ msgstr "valeur invalide de 'sparse' '%s'"

		#, fuzzy
		#~| msgid "unable to create '%s'"
		#~ msgid "unable to resolve '%s'"
		#~ msgstr "impossible de créer '%s'"

		#~ msgid "unmerged:   %s"
		#~ msgstr "non fus. :  %s"
		EOF
	)
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=error msg="[po/fr.po]    2 translated messages."
level=error msg="[po/fr.po]    too many obsolete entries (3) in comments, please remove them"
level=error msg="[po/fr.po]    remove lines that start with '#~| msgid', for they are not compatible with gettext 0.14"
ERROR: check-po command failed
EOF

test_expect_success "show gettext 0.14 incompatible errors" '
	test_must_fail git -C workdir $HELPER check-po $POT_NO --report-file-locations=none fr >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
