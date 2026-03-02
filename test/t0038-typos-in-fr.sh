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
	cat >expect <<-EOF &&
		------------------------------------------------------------------------------
		level=error msg="[po/fr.po]    5282 translated messages."
		level=error msg="[po/fr.po]    too many obsolete entries (178) in comments, please remove them"
		------------------------------------------------------------------------------
		level=error msg="[po/fr.po]    mismatched patterns: refs/"
		level=error msg="[po/fr.po]    >> msgid: The destination you provided is not a full refname (i.e.,"
		level=error msg="[po/fr.po]    starting with \"refs/\"). We tried to guess what you meant by:"
		level=error msg="[po/fr.po]"
		level=error msg="[po/fr.po]    - Looking for a ref that matches ${SQ}%s${SQ} on the remote side."
		level=error msg="[po/fr.po]    - Checking if the <src> being pushed (${SQ}%s${SQ})"
		level=error msg="[po/fr.po]     is a ref in \"refs/{heads,tags}/\". If so we add a corresponding"
		level=error msg="[po/fr.po]     refs/{heads,tags}/ prefix on the remote side."
		level=error msg="[po/fr.po]"
		level=error msg="[po/fr.po]    Neither worked, so we gave up. You must fully qualify the ref."
		level=error msg="[po/fr.po]    >> msgstr: La destination que vous avez fournie n${SQ}est pas un nom de référence complète"
		level=error msg="[po/fr.po]    (c${SQ}est-à-dire commençant par \"ref/\"). Essai d${SQ}approximation par\u00a0:"
		level=error msg="[po/fr.po]"
		level=error msg="[po/fr.po]    - Recherche d${SQ}une référence qui correspond à ${SQ}%s${SQ} sur le serveur distant."
		level=error msg="[po/fr.po]    - Vérification si la <source> en cours de poussée (${SQ}%s${SQ})"
		level=error msg="[po/fr.po]     est une référence dans \"refs/{heads,tags}/\". Si oui, ajout du préfixe"
		level=error msg="[po/fr.po]     refs/{heads,tags}/ correspondant du côté distant."
		level=error msg="[po/fr.po]"
		level=error msg="[po/fr.po]    Aucune n${SQ}a fonctionné, donc abandon. Veuillez spécifier une référence totalement qualifiée."
		level=error msg="[po/fr.po]"
		ERROR: check-po command failed
	EOF
	test_cmp expect actual
'

test_done
