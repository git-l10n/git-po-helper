#!/bin/sh

test_description="check typos in fr.po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn --report-file-locations=none"
POT_NO="--pot-file=no"

test_expect_success "checkout master branch" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout master
'

test_expect_success "still has typos in master branch" '
	test_must_fail git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error po/fr.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cat >expect <<-\EOF &&
	ℹ️ Syntax check with msgfmt
	 INFO [fr.po] 5282 translated messages.
	❌ Obsolete #~ entries
	 ERROR [fr.po] you have 178 obsolete entries, please remove them
	❌ msgid/msgstr pattern check
	 ERROR [fr.po] mismatched patterns: refs/
	 ERROR [fr.po] >> msgid: The destination you provided is not a full refname (i.e.,
	 ERROR [fr.po] starting with "refs/"). We tried to guess what you meant by:
	 ERROR [fr.po]
	 ERROR [fr.po] - Looking for a ref that matches '\''%s'\'' on the remote side.
	 ERROR [fr.po] - Checking if the <src> being pushed ('\''%s'\'')
	 ERROR [fr.po] is a ref in "refs/{heads,tags}/". If so we add a corresponding
	 ERROR [fr.po] refs/{heads,tags}/ prefix on the remote side.
	 ERROR [fr.po]
	 ERROR [fr.po] Neither worked, so we gave up. You must fully qualify the ref.
	 ERROR [fr.po] >> msgstr: La destination que vous avez fournie n'\''est pas un nom de référence complète
	 ERROR [fr.po] (c'\''est-à-dire commençant par "ref/"). Essai d'\''approximation par :
	 ERROR [fr.po]
	 ERROR [fr.po] - Recherche d'\''une référence qui correspond à '\''%s'\'' sur le serveur distant.
	 ERROR [fr.po] - Vérification si la <source> en cours de poussée ('\''%s'\'')
	 ERROR [fr.po] est une référence dans "refs/{heads,tags}/". Si oui, ajout du préfixe
	 ERROR [fr.po] refs/{heads,tags}/ correspondant du côté distant.
	 ERROR [fr.po]
	 ERROR [fr.po] Aucune n'\''a fonctionné, donc abandon. Veuillez spécifier une référence totalement qualifiée.
	 ERROR [fr.po]
	ERROR: check-po command failed
	EOF
	test_cmp expect actual
'

test_done
