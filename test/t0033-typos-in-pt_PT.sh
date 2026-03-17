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
ℹ️ Syntax check with msgfmt
 INFO [pt_PT.po] 2876 translated messages, 1320 fuzzy translations, 842 untranslated messages.
❌ Obsolete #~ entries
 ERROR [pt_PT.po] you have 225 obsolete entries, please remove them
⚠️ msgid/msgstr pattern check
 WARNING [pt_PT.po] mismatched patterns: --contains, --no-contains
 WARNING [pt_PT.po] >> msgid: --no-contains option is only allowed in list mode
 WARNING [pt_PT.po] >> msgstr: a opção --contains só é permitida no modo lista
 WARNING [pt_PT.po]
 WARNING [pt_PT.po] mismatched patterns: --no-write-bitmap-index
 WARNING [pt_PT.po] >> msgid: Incremental repacks are incompatible with bitmap indexes. Use
 WARNING [pt_PT.po] --no-write-bitmap-index or disable the pack.writebitmaps configuration.
 WARNING [pt_PT.po] >> msgstr: Repacks incrementais são incompatíveis com bitmap indexes. Usa
 WARNING [pt_PT.po] --no-write-bitmap-índex ou desative a configuração pack.writebitmaps.
 WARNING [pt_PT.po]
 WARNING [pt_PT.po] mismatched patterns: $n
 WARNING [pt_PT.po] >> msgid: The commit message #${n} will be skipped:
 WARNING [pt_PT.po] >> msgstr: A mensagem de commit nº${n} será ignorada:
 WARNING [pt_PT.po]
 WARNING [pt_PT.po] mismatched patterns: $n
 WARNING [pt_PT.po] >> msgid: This is the commit message #${n}:
 WARNING [pt_PT.po] >> msgstr: Esta é a mensagem de commit nº${n}:
 WARNING [pt_PT.po]
 WARNING [pt_PT.po] mismatched patterns: fetch.ouput, fetch.output
 WARNING [pt_PT.po] >> msgid: configuration fetch.output contains invalid value %s
 WARNING [pt_PT.po] >> msgstr: a configuração fetch.ouput contém o valor inválido %s
 WARNING [pt_PT.po]
 WARNING [pt_PT.po] mismatched patterns: git-mailsplit
 WARNING [pt_PT.po] >> msgid: pass --keep-cr flag to git-mailsplit for mbox format
 WARNING [pt_PT.po] >> msgstr: passar a opção --keep-cr ao gitmailsplit para formato de mbox
 WARNING [pt_PT.po]
 WARNING [pt_PT.po] mismatched patterns: %%(algn), %%(align)
 WARNING [pt_PT.po] >> msgid: positive width expected with the %%(align) atom
 WARNING [pt_PT.po] >> msgstr: largura positiva esperada com o átomo %%(algn)
 WARNING [pt_PT.po]
ERROR: check-po command failed
EOF

test_expect_success "check typos in pt_PT.po" '
	test_must_fail git -C workdir $HELPER check-po $POT_NO \
		po/pt_PT.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "no typos in master branch" '
	git -C workdir checkout master &&
	git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error \
		po/pt_PT.po
'

test_done
