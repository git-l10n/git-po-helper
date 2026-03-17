#!/bin/sh

test_description="check typos in es.po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn --report-file-locations=none"
POT_NO="--pot-file=no"

test_expect_success "checkout po-2.31.1" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1
'

cat >expect <<-\EOF
ℹ️ Syntax check with msgfmt
 INFO [es.po] 5104 translated messages.
❌ Obsolete #~ entries
 ERROR [es.po] you have 235 obsolete entries, please remove them
⚠️ msgid/msgstr pattern check
 WARNING [es.po] mismatched patterns: herramienta.cmd
 WARNING [es.po] >> msgid: '%s': path for unsupported man viewer.
 WARNING [es.po] Please consider using 'man.<tool>.cmd' instead.
 WARNING [es.po] >> msgstr: '%s': ruta para el visualizador del manual no soportada.
 WARNING [es.po] Por favor considere usar 'man.<herramienta.cmd'.
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: --porcelain=2, --procelain=2
 WARNING [es.po] >> msgid: 'git status --porcelain=2' failed in submodule %s
 WARNING [es.po] >> msgstr: 'git status --procelain=2' falló en el submódulo %s
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: --dir-diff, --dirty-diff
 WARNING [es.po] >> msgid: --dir-diff is incompatible with --no-index
 WARNING [es.po] >> msgstr: --dirty-diff es incompatible con --no-index
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: extensions.partialClone, extensions.partialclone
 WARNING [es.po] >> msgid: --filter can only be used with the remote configured in extensions.partialclone
 WARNING [es.po] >> msgstr: --filter solo puede ser usado con el remoto configurado en extensions.partialClone
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: --merge-base, --merge-baseson
 WARNING [es.po] >> msgid: --stdin and --merge-base are mutually exclusive
 WARNING [es.po] >> msgstr: --stdin and --merge-baseson mutuamente exclusivas
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: --porcelain=2, --procelain=2
 WARNING [es.po] >> msgid: Could not run 'git status --porcelain=2' in submodule %s
 WARNING [es.po] >> msgstr: No se pudo ejecutar 'git status --procelain=2' en el submódulo %s
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: --shallow-exclude, --shalow-exclude
 WARNING [es.po] >> msgid: Server does not support --shallow-exclude
 WARNING [es.po] >> msgstr: El servidor no soporta --shalow-exclude
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: --shallow-since, --shalow-since
 WARNING [es.po] >> msgid: Server does not support --shallow-since
 WARNING [es.po] >> msgstr: El servidor no soporta --shalow-since
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: refs/heads/, refs/tags/
 WARNING [es.po] >> msgid: The <src> part of the refspec is a blob object.
 WARNING [es.po] Did you mean to tag a new blob by pushing to
 WARNING [es.po] '%s:refs/tags/%s'?
 WARNING [es.po] >> msgstr: La parte <src> del refspec es un objeto blob.
 WARNING [es.po] ¿Quisiste crear un tag nuevo mediante un push a
 WARNING [es.po] '%s:refs/heads/%s'?
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: refs/heads/, refs/tags/
 WARNING [es.po] >> msgid: The <src> part of the refspec is a tree object.
 WARNING [es.po] Did you mean to tag a new tree by pushing to
 WARNING [es.po] '%s:refs/tags/%s'?
 WARNING [es.po] >> msgstr: La parte <src> del refspec es un objeto tree.
 WARNING [es.po] ¿Quisiste crear un tag nuevo mediante un push a
 WARNING [es.po] '%s:refs/heads/%s'?
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: ---smtp-debug, --smtp-debug
 WARNING [es.po] >> msgid: Unable to initialize SMTP properly. Check config and use --smtp-debug.
 WARNING [es.po] >> msgstr: No es posible inicializar SMTP adecuadamente. Verificar config y usar ---smtp-debug.
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: --allow-empty, --alow-empty
 WARNING [es.po] >> msgid: You asked to amend the most recent commit, but doing so would make
 WARNING [es.po] it empty. You can repeat your command with --allow-empty, or you can
 WARNING [es.po] remove the commit entirely with "git reset HEAD^".
 WARNING [es.po]
 WARNING [es.po] >> msgstr: Has solicitado un amend en tu commit más reciente, pero hacerlo lo
 WARNING [es.po] vaciaría. Puedes repetir el comando con --alow-empty, o puedes eliminar
 WARNING [es.po] el commit completamente con "git reset HEAD^".
 WARNING [es.po]
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: --filter, usar--filter
 WARNING [es.po] >> msgid: cannot use --filter without --stdout
 WARNING [es.po] >> msgstr: no se puede usar--filter sin --stdout
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: dimmed_zebra
 WARNING [es.po] >> msgid: color moved setting must be one of 'no', 'default', 'blocks', 'zebra', 'dimmed-zebra', 'plain'
 WARNING [es.po] >> msgstr: opción de color tiene que ser una de 'no', 'default', 'blocks', 'zebra', 'dimmed_zebra', 'plain'
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: refs/remotes/<...>/HEAD, refs/remotos/<...>/HEAD
 WARNING [es.po] >> msgid: delete refs/remotes/<name>/HEAD
 WARNING [es.po] >> msgstr: borrar refs/remotos/<nombre>/HEAD
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: gc.logexpirity, gc.logexpiry
 WARNING [es.po] >> msgid: failed to parse gc.logexpiry value %s
 WARNING [es.po] >> msgstr: falló al analizar valor %s de gc.logexpirity
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: format.headers, formate.headers
 WARNING [es.po] >> msgid: format.headers without value
 WARNING [es.po] >> msgstr: formate.headers. sin valor
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: git-apply
 WARNING [es.po] >> msgid: passed to 'git apply'
 WARNING [es.po] >> msgstr: pasado a 'git-apply'
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: git-upload-archive, git-upload-archivo
 WARNING [es.po] >> msgid: path to the remote git-upload-archive command
 WARNING [es.po] >> msgstr: ruta para el comando git-upload-archivo remoto
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: %%(%s)
 WARNING [es.po] >> msgid: positive value expected '%s' in %%(%s)
 WARNING [es.po] >> msgstr: valor positivo esperado '%s' en %% (%s)
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: --sateged, --staged
 WARNING [es.po] >> msgid: repository has been updated, but unable to write
 WARNING [es.po] new_index file. Check that disk is not full and quota is
 WARNING [es.po] not exceeded, and then "git restore --staged :/" to recover.
 WARNING [es.po] >> msgstr: el repositorio ha sido actualizado, pero no se pudo escribir el archivo
 WARNING [es.po] new_index. Verifique que el disco no este lleno y la quota no ha
 WARNING [es.po] sido superada, y luego "git restore --sateged :/" para recuperar.
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: submodule--helper, subomdule--helper
 WARNING [es.po] >> msgid: submodule--helper print-default-remote takes no arguments
 WARNING [es.po] >> msgstr: subomdule--helper print-default-remote no toma argumentos
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: --raw, --stat
 WARNING [es.po] >> msgid: synonym for '-p --raw'
 WARNING [es.po] >> msgstr: sinónimo para '-p --stat'
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: --abort, s--abort
 WARNING [es.po] >> msgid: try "git cherry-pick (--continue | %s--abort | --quit)"
 WARNING [es.po] >> msgstr: intenta "git cherry-pick (--continue | --quit | %s --abort)"
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: --abort, s--abort
 WARNING [es.po] >> msgid: try "git revert (--continue | %s--abort | --quit)"
 WARNING [es.po] >> msgstr: intenta "git revert (--continue | --quit | %s --abort)"
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: load_cache_entires, load_cache_entries
 WARNING [es.po] >> msgid: unable to join load_cache_entries thread: %s
 WARNING [es.po] >> msgstr: no es posible unir hilo load_cache_entires: %s
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: %%(subject), %%(sujeto)
 WARNING [es.po] >> msgid: unrecognized %%(subject) argument: %s
 WARNING [es.po] >> msgstr: argumento %%(sujeto) no reconocido: %s
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: --reference, usa--reference
 WARNING [es.po] >> msgid: use --reference only while cloning
 WARNING [es.po] >> msgstr: usa--reference solamente si estás clonado
 WARNING [es.po]
 WARNING [es.po] mismatched patterns: --group, --group=trailer
 WARNING [es.po] >> msgid: using --group=trailer with stdin is not supported
 WARNING [es.po] >> msgstr: el uso de --group = trailer con stdin no es compatible
 WARNING [es.po]
ERROR: check-po command failed
EOF

test_expect_success "check typos in es.po of git 2.31.1" '
	test_must_fail git -C workdir $HELPER check-po $POT_NO \
		po/es.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
ℹ️ Syntax check with msgfmt
 INFO [es.po] 5210 translated messages.
❌ Obsolete #~ entries
 ERROR [es.po] you have 125 obsolete entries, please remove them
❌ msgid/msgstr pattern check
 ERROR [es.po] mismatched patterns: refs/remotes/<...>/HEAD, refs/remotos/<...>/HEAD
 ERROR [es.po] >> msgid: delete refs/remotes/<name>/HEAD
 ERROR [es.po] >> msgstr: borrar refs/remotos/<nombre>/HEAD
 ERROR [es.po]
 ERROR [es.po] mismatched patterns: refs/preferch/, refs/prefetch/
 ERROR [es.po] >> msgid: modify the refspec to place all refs within refs/prefetch/
 ERROR [es.po] >> msgstr: modificar el refspec para colocar todas las referencias en refs/preferch/
 ERROR [es.po]
ERROR: check-po command failed
EOF

test_expect_success "typos in master branch" '
	git -C workdir checkout master &&
	test_must_fail git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error \
		po/es.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
