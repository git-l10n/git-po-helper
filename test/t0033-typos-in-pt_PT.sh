#!/bin/sh

test_description="check typos in pt_PT.po"

. ./lib/sharness.sh

HELPER="git-po-helper --no-gettext-back-compatible"

test_expect_success "setup" '
	mkdir po &&
	touch po/git.pot &&
	cp "${PO_HELPER_TEST_REPOSITORY}/po/pt_PT.po" po
'

cat >expect <<-\EOF
level=info msg="[po/pt_PT.po]    2876 translated messages, 1320 fuzzy translations, 842 untranslated messages."
level=warning msg="[po/pt_PT.po]    mismatch variable names: --contains, --no-contains"
level=warning msg="[po/pt_PT.po]    >> msgid: --no-contains option is only allowed in list mode"
level=warning msg="[po/pt_PT.po]    >> msgstr: a opção --contains só é permitida no modo lista"
level=warning
level=warning msg="[po/pt_PT.po]    mismatch variable names: --no-write-bitmap-index"
level=warning msg="[po/pt_PT.po]    >> msgid: Incremental repacks are incompatible with bitmap indexes. Use\n--no-write-bitmap-index or disable the pack.writebitmaps configuration."
level=warning msg="[po/pt_PT.po]    >> msgstr: Repacks incrementais são incompatíveis com bitmap indexes. Usa\n--no-write-bitmap-índex ou desative a configuração pack.writebitmaps."
level=warning
level=warning msg="[po/pt_PT.po]    mismatch variable names: $n"
level=warning msg="[po/pt_PT.po]    >> msgid: The commit message #${n} will be skipped:"
level=warning msg="[po/pt_PT.po]    >> msgstr: A mensagem de commit nº${n} será ignorada:"
level=warning
level=warning msg="[po/pt_PT.po]    mismatch variable names: $n"
level=warning msg="[po/pt_PT.po]    >> msgid: This is the commit message #${n}:"
level=warning msg="[po/pt_PT.po]    >> msgstr: Esta é a mensagem de commit nº${n}:"
level=warning
level=warning msg="[po/pt_PT.po]    mismatch variable names: fetch.ouput, fetch.output"
level=warning msg="[po/pt_PT.po]    >> msgid: configuration fetch.output contains invalid value %s"
level=warning msg="[po/pt_PT.po]    >> msgstr: a configuração fetch.ouput contém o valor inválido %s"
level=warning
level=warning msg="[po/pt_PT.po]    mismatch variable names: git-mailsplit"
level=warning msg="[po/pt_PT.po]    >> msgid: pass --keep-cr flag to git-mailsplit for mbox format"
level=warning msg="[po/pt_PT.po]    >> msgstr: passar a opção --keep-cr ao gitmailsplit para formato de mbox"
level=warning
EOF

test_expect_success "check typos in pt_PT.po" '
	$HELPER check-po pt_PT >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
