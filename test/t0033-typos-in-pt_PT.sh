#!/bin/sh

test_description="check typos in pt_PT.po"

. ./lib/sharness.sh

test_expect_success "setup" '
	mkdir po &&
	touch po/git.pot &&
	cp ../examples/pt_PT.po po
'

cat >expect <<-\EOF
[po/pt_PT.po] 2876 translated messages, 1320 fuzzy translations, 842 untranslated messages.
level=warning msg="mismatch variable names: fetch.output"
level=warning msg=">> msgid: configuration fetch.output contains invalid value %s"
level=warning msg=">> msgstr: a configuração fetch.ouput contém o valor inválido %s"
level=warning
EOF

test_expect_success "check typos in pt_PT.po" '
	git-po-helper check-po pt_PT >actual 2>&1 &&
	test_cmp expect actual
'

test_done
