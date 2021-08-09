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
level=warning msg="mismatch variable names: herramienta.cmd"
level=warning msg=">> msgid: '%s': path for unsupported man viewer.\nPlease consider using 'man.<tool>.cmd' instead."
level=warning msg=">> msgstr: '%s': ruta para el visualizador del manual no soportada.\nPor favor considere usar 'man.<herramienta.cmd'."
level=warning
level=warning msg="mismatch variable names: --porcelain=, --procelain="
level=warning msg=">> msgid: 'git status --porcelain=2' failed in submodule %s"
level=warning msg=">> msgstr: 'git status --procelain=2' falló en el submódulo %s"
level=warning
level=warning msg="mismatch variable names: --dir-diff, --dirty-diff"
level=warning msg=">> msgid: --dir-diff is incompatible with --no-index"
level=warning msg=">> msgstr: --dirty-diff es incompatible con --no-index"
level=warning
level=warning msg="mismatch variable names: extensions.partialClone, extensions.partialclone"
level=warning msg=">> msgid: --filter can only be used with the remote configured in extensions.partialclone"
level=warning msg=">> msgstr: --filter solo puede ser usado con el remoto configurado en extensions.partialClone"
level=warning
level=warning msg="mismatch variable names: --merge-base, --merge-baseson"
level=warning msg=">> msgid: --stdin and --merge-base are mutually exclusive"
level=warning msg=">> msgstr: --stdin and --merge-baseson mutuamente exclusivas"
level=warning
level=warning msg="mismatch variable names: --porcelain=, --procelain="
level=warning msg=">> msgid: Could not run 'git status --porcelain=2' in submodule %s"
level=warning msg=">> msgstr: No se pudo ejecutar 'git status --procelain=2' en el submódulo %s"
level=warning
level=warning msg="mismatch variable names: --shallow-exclude, --shalow-exclude"
level=warning msg=">> msgid: Server does not support --shallow-exclude"
level=warning msg=">> msgstr: El servidor no soporta --shalow-exclude"
level=warning
level=warning msg="mismatch variable names: --shallow-since, --shalow-since"
level=warning msg=">> msgid: Server does not support --shallow-since"
level=warning msg=">> msgstr: El servidor no soporta --shalow-since"
level=warning
level=warning msg="mismatch variable names: --allow-empty, --alow-empty"
level=warning msg=">> msgid: You asked to amend the most recent commit, but doing so would make\nit empty. You can repeat your command with --allow-empty, or you can\nremove the commit entirely with \"git reset HEAD^\".\n"
level=warning msg=">> msgstr: Has solicitado un amend en tu commit más reciente, pero hacerlo lo\nvaciaría. Puedes repetir el comando con --alow-empty, o puedes eliminar\nel commit completamente con \"git reset HEAD^\".\n"
level=warning
level=warning msg="mismatch variable names: dimmed_zebra"
level=warning msg=">> msgid: color moved setting must be one of 'no', 'default', 'blocks', 'zebra', 'dimmed-zebra', 'plain'"
level=warning msg=">> msgstr: opción de color tiene que ser una de 'no', 'default', 'blocks', 'zebra', 'dimmed_zebra', 'plain'"
level=warning
level=warning msg="mismatch variable names: gc.logexpirity, gc.logexpiry"
level=warning msg=">> msgid: failed to parse gc.logexpiry value %s"
level=warning msg=">> msgstr: falló al analizar valor %s de gc.logexpirity"
level=warning
level=warning msg="mismatch variable names: format.headers, formate.headers"
level=warning msg=">> msgid: format.headers without value"
level=warning msg=">> msgstr: formate.headers. sin valor"
level=warning
level=warning msg="mismatch variable names: submodule--helper"
level=warning msg=">> msgid: submodule--helper print-default-remote takes no arguments"
level=warning msg=">> msgstr: subomdule--helper print-default-remote no toma argumentos"
level=warning
level=warning msg="mismatch variable names: git-apply"
level=warning msg=">> msgid: passed to 'git apply'"
level=warning msg=">> msgstr: pasado a 'git-apply'"
level=warning
level=warning msg="mismatch variable names: s--abort"
level=warning msg=">> msgid: try \"git cherry-pick (--continue | %s--abort | --quit)\""
level=warning msg=">> msgstr: intenta \"git cherry-pick (--continue | --quit | %s --abort)\""
level=warning
level=warning msg="mismatch variable names: --sateged, --staged"
level=warning msg=">> msgid: repository has been updated, but unable to write\nnew_index file. Check that disk is not full and quota is\nnot exceeded, and then \"git restore --staged :/\" to recover."
level=warning msg=">> msgstr: el repositorio ha sido actualizado, pero no se pudo escribir el archivo\nnew_index. Verifique que el disco no este lleno y la quota no ha\nsido superada, y luego \"git restore --sateged :/\" para recuperar."
level=warning
level=warning msg="mismatch variable names: s--abort"
level=warning msg=">> msgid: try \"git revert (--continue | %s--abort | --quit)\""
level=warning msg=">> msgstr: intenta \"git revert (--continue | --quit | %s --abort)\""
level=warning
level=warning msg="mismatch variable names: --raw, --stat"
level=warning msg=">> msgid: synonym for '-p --raw'"
level=warning msg=">> msgstr: sinónimo para '-p --stat'"
level=warning
level=warning msg="mismatch variable names: --dirstat=archivos, --dirstat=files"
level=warning msg=">> msgid: synonym for --dirstat=files,param1,param2..."
level=warning msg=">> msgstr: sinonimo para --dirstat=archivos,param1,param2..."
level=warning
level=warning msg="mismatch variable names: load_cache_entires, load_cache_entries"
level=warning msg=">> msgid: unable to join load_cache_entries thread: %s"
level=warning msg=">> msgstr: no es posible unir hilo load_cache_entires: %s"
level=warning
level=warning msg="mismatch variable names: --group, --group=trailer"
level=warning msg=">> msgid: using --group=trailer with stdin is not supported"
level=warning msg=">> msgstr: el uso de --group = trailer con stdin no es compatible"
level=warning
EOF

test_expect_failure "check typos in es.po" '
	git-po-helper check-po es >actual 2>&1 &&
	test_cmp expect actual
'

test_done
