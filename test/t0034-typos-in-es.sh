#!/bin/sh

test_description="check typos in es.po"

. ./lib/sharness.sh

test_expect_success "setup" '
	mkdir po &&
	touch po/git.pot &&
	cp ../examples/es.po po
'

cat >expect <<-\EOF
[po/es.po]    5204 translated messages.
level=warning msg="mismatch variable names: extensions.partialclone"
level=warning msg=">> msgid: --filter can only be used with the remote configured in extensions.partialclone"
level=warning msg=">> msgstr: --filter solo puede ser usado con el remoto configurado en extensions.partialClone"
level=warning
level=warning msg="mismatch variable names: gc.logexpiry"
level=warning msg=">> msgid: failed to parse gc.logexpiry value %s"
level=warning msg=">> msgstr: falló al analizar valor %s de gc.logexpirity"
level=warning
level=warning msg="mismatch variable names: format.headers"
level=warning msg=">> msgid: format.headers without value"
level=warning msg=">> msgstr: formate.headers. sin valor"
level=warning
level=warning msg="mismatch variable names: submodule--helper"
level=warning msg=">> msgid: submodule--helper print-default-remote takes no arguments"
level=warning msg=">> msgstr: subomdule--helper print-default-remote no toma argumentos"
level=warning
level=warning msg="mismatch variable names: s--abort"
level=warning msg=">> msgid: try \"git cherry-pick (--continue | %s--abort | --quit)\""
level=warning msg=">> msgstr: intenta \"git cherry-pick (--continue | --quit | %s --abort)\""
level=warning
level=warning msg="mismatch variable names: s--abort"
level=warning msg=">> msgid: try \"git revert (--continue | %s--abort | --quit)\""
level=warning msg=">> msgstr: intenta \"git revert (--continue | --quit | %s --abort)\""
level=warning
level=warning msg="mismatch variable names: load_cache_entries"
level=warning msg=">> msgid: unable to join load_cache_entries thread: %s"
level=warning msg=">> msgstr: no es posible unir hilo load_cache_entires: %s"
level=warning
EOF

test_expect_success "check typos in es.po" '
	git-po-helper check-po es >actual 2>&1 &&
	test_cmp expect actual
'

test_done
