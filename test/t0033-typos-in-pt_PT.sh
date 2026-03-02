#!/bin/sh

test_description="check typos in pt_PT.po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn --report-file-locations=none"
POT_NO="--pot-file=no"

test_expect_success "checkout po-2.31.1" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=error msg="[po/pt_PT.po]    2876 translated messages, 1320 fuzzy translations, 842 untranslated messages."
level=error msg="[po/pt_PT.po]    too many obsolete entries (225) in comments, please remove them"
------------------------------------------------------------------------------
level=warning msg="[po/pt_PT.po]    mismatched patterns: --contains, --no-contains"
level=warning msg="[po/pt_PT.po]    >> msgid: --no-contains option is only allowed in list mode"
level=warning msg="[po/pt_PT.po]    >> msgstr: a opção --contains só é permitida no modo lista"
level=warning msg="[po/pt_PT.po]"
level=warning msg="[po/pt_PT.po]    mismatched patterns: --no-write-bitmap-index"
level=warning msg="[po/pt_PT.po]    >> msgid: Incremental repacks are incompatible with bitmap indexes. Use"
level=warning msg="[po/pt_PT.po]    --no-write-bitmap-index or disable the pack.writebitmaps configuration."
level=warning msg="[po/pt_PT.po]    >> msgstr: Repacks incrementais são incompatíveis com bitmap indexes. Usa"
level=warning msg="[po/pt_PT.po]    --no-write-bitmap-índex ou desative a configuração pack.writebitmaps."
level=warning msg="[po/pt_PT.po]"
level=warning msg="[po/pt_PT.po]    mismatched patterns: $n"
level=warning msg="[po/pt_PT.po]    >> msgid: The commit message #${n} will be skipped:"
level=warning msg="[po/pt_PT.po]    >> msgstr: A mensagem de commit nº${n} será ignorada:"
level=warning msg="[po/pt_PT.po]"
level=warning msg="[po/pt_PT.po]    mismatched patterns: $n"
level=warning msg="[po/pt_PT.po]    >> msgid: This is the commit message #${n}:"
level=warning msg="[po/pt_PT.po]    >> msgstr: Esta é a mensagem de commit nº${n}:"
level=warning msg="[po/pt_PT.po]"
level=warning msg="[po/pt_PT.po]    mismatched patterns: fetch.ouput, fetch.output"
level=warning msg="[po/pt_PT.po]    >> msgid: configuration fetch.output contains invalid value %s"
level=warning msg="[po/pt_PT.po]    >> msgstr: a configuração fetch.ouput contém o valor inválido %s"
level=warning msg="[po/pt_PT.po]"
level=warning msg="[po/pt_PT.po]    mismatched patterns: git-mailsplit"
level=warning msg="[po/pt_PT.po]    >> msgid: pass --keep-cr flag to git-mailsplit for mbox format"
level=warning msg="[po/pt_PT.po]    >> msgstr: passar a opção --keep-cr ao gitmailsplit para formato de mbox"
level=warning msg="[po/pt_PT.po]"
level=warning msg="[po/pt_PT.po]    mismatched patterns: %%(algn), %%(align)"
level=warning msg="[po/pt_PT.po]    >> msgid: positive width expected with the %%(align) atom"
level=warning msg="[po/pt_PT.po]    >> msgstr: largura positiva esperada com o átomo %%(algn)"
level=warning msg="[po/pt_PT.po]"
ERROR: check-po command failed
EOF

test_expect_success "check typos in pt_PT.po" '
	test_must_fail git -C workdir $HELPER check-po $POT_NO pt_PT >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "no typos in master branch" '
	git -C workdir checkout master &&
	git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error pt_PT
'

test_done
