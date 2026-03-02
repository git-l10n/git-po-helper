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
------------------------------------------------------------------------------
level=error msg="[po/es.po]    5104 translated messages."
level=error msg="[po/es.po]    too many obsolete entries (235) in comments, please remove them"
------------------------------------------------------------------------------
level=warning msg="[po/es.po]    mismatched patterns: herramienta.cmd"
level=warning msg="[po/es.po]    >> msgid: '%s': path for unsupported man viewer."
level=warning msg="[po/es.po]    Please consider using 'man.<tool>.cmd' instead."
level=warning msg="[po/es.po]    >> msgstr: '%s': ruta para el visualizador del manual no soportada."
level=warning msg="[po/es.po]    Por favor considere usar 'man.<herramienta.cmd'."
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: --porcelain=2, --procelain=2"
level=warning msg="[po/es.po]    >> msgid: 'git status --porcelain=2' failed in submodule %s"
level=warning msg="[po/es.po]    >> msgstr: 'git status --procelain=2' falló en el submódulo %s"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: --dir-diff, --dirty-diff"
level=warning msg="[po/es.po]    >> msgid: --dir-diff is incompatible with --no-index"
level=warning msg="[po/es.po]    >> msgstr: --dirty-diff es incompatible con --no-index"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: extensions.partialClone, extensions.partialclone"
level=warning msg="[po/es.po]    >> msgid: --filter can only be used with the remote configured in extensions.partialclone"
level=warning msg="[po/es.po]    >> msgstr: --filter solo puede ser usado con el remoto configurado en extensions.partialClone"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: --merge-base, --merge-baseson"
level=warning msg="[po/es.po]    >> msgid: --stdin and --merge-base are mutually exclusive"
level=warning msg="[po/es.po]    >> msgstr: --stdin and --merge-baseson mutuamente exclusivas"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: --porcelain=2, --procelain=2"
level=warning msg="[po/es.po]    >> msgid: Could not run 'git status --porcelain=2' in submodule %s"
level=warning msg="[po/es.po]    >> msgstr: No se pudo ejecutar 'git status --procelain=2' en el submódulo %s"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: --shallow-exclude, --shalow-exclude"
level=warning msg="[po/es.po]    >> msgid: Server does not support --shallow-exclude"
level=warning msg="[po/es.po]    >> msgstr: El servidor no soporta --shalow-exclude"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: --shallow-since, --shalow-since"
level=warning msg="[po/es.po]    >> msgid: Server does not support --shallow-since"
level=warning msg="[po/es.po]    >> msgstr: El servidor no soporta --shalow-since"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: refs/heads/, refs/tags/"
level=warning msg="[po/es.po]    >> msgid: The <src> part of the refspec is a blob object."
level=warning msg="[po/es.po]    Did you mean to tag a new blob by pushing to"
level=warning msg="[po/es.po]    '%s:refs/tags/%s'?"
level=warning msg="[po/es.po]    >> msgstr: La parte <src> del refspec es un objeto blob."
level=warning msg="[po/es.po]    ¿Quisiste crear un tag nuevo mediante un push a"
level=warning msg="[po/es.po]    '%s:refs/heads/%s'?"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: refs/heads/, refs/tags/"
level=warning msg="[po/es.po]    >> msgid: The <src> part of the refspec is a tree object."
level=warning msg="[po/es.po]    Did you mean to tag a new tree by pushing to"
level=warning msg="[po/es.po]    '%s:refs/tags/%s'?"
level=warning msg="[po/es.po]    >> msgstr: La parte <src> del refspec es un objeto tree."
level=warning msg="[po/es.po]    ¿Quisiste crear un tag nuevo mediante un push a"
level=warning msg="[po/es.po]    '%s:refs/heads/%s'?"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: ---smtp-debug, --smtp-debug"
level=warning msg="[po/es.po]    >> msgid: Unable to initialize SMTP properly. Check config and use --smtp-debug."
level=warning msg="[po/es.po]    >> msgstr: No es posible inicializar SMTP adecuadamente. Verificar config y usar ---smtp-debug."
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: --allow-empty, --alow-empty"
level=warning msg="[po/es.po]    >> msgid: You asked to amend the most recent commit, but doing so would make"
level=warning msg="[po/es.po]    it empty. You can repeat your command with --allow-empty, or you can"
level=warning msg="[po/es.po]    remove the commit entirely with \"git reset HEAD^\"."
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    >> msgstr: Has solicitado un amend en tu commit más reciente, pero hacerlo lo "
level=warning msg="[po/es.po]    vaciaría. Puedes repetir el comando con --alow-empty, o puedes eliminar"
level=warning msg="[po/es.po]    el commit completamente con \"git reset HEAD^\"."
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: --filter, usar--filter"
level=warning msg="[po/es.po]    >> msgid: cannot use --filter without --stdout"
level=warning msg="[po/es.po]    >> msgstr: no se puede usar--filter sin --stdout"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: dimmed_zebra"
level=warning msg="[po/es.po]    >> msgid: color moved setting must be one of 'no', 'default', 'blocks', 'zebra', 'dimmed-zebra', 'plain'"
level=warning msg="[po/es.po]    >> msgstr: opción de color tiene que ser una de 'no', 'default', 'blocks', 'zebra', 'dimmed_zebra', 'plain'"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: refs/remotes/<...>/HEAD, refs/remotos/<...>/HEAD"
level=warning msg="[po/es.po]    >> msgid: delete refs/remotes/<name>/HEAD"
level=warning msg="[po/es.po]    >> msgstr: borrar refs/remotos/<nombre>/HEAD"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: gc.logexpirity, gc.logexpiry"
level=warning msg="[po/es.po]    >> msgid: failed to parse gc.logexpiry value %s"
level=warning msg="[po/es.po]    >> msgstr: falló al analizar valor %s de gc.logexpirity"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: format.headers, formate.headers"
level=warning msg="[po/es.po]    >> msgid: format.headers without value"
level=warning msg="[po/es.po]    >> msgstr: formate.headers. sin valor"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: git-apply"
level=warning msg="[po/es.po]    >> msgid: passed to 'git apply'"
level=warning msg="[po/es.po]    >> msgstr: pasado a 'git-apply'"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: git-upload-archive, git-upload-archivo"
level=warning msg="[po/es.po]    >> msgid: path to the remote git-upload-archive command"
level=warning msg="[po/es.po]    >> msgstr: ruta para el comando git-upload-archivo remoto"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: %%(%s)"
level=warning msg="[po/es.po]    >> msgid: positive value expected '%s' in %%(%s)"
level=warning msg="[po/es.po]    >> msgstr: valor positivo esperado '%s' en %% (%s)"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: --sateged, --staged"
level=warning msg="[po/es.po]    >> msgid: repository has been updated, but unable to write"
level=warning msg="[po/es.po]    new_index file. Check that disk is not full and quota is"
level=warning msg="[po/es.po]    not exceeded, and then \"git restore --staged :/\" to recover."
level=warning msg="[po/es.po]    >> msgstr: el repositorio ha sido actualizado, pero no se pudo escribir el archivo"
level=warning msg="[po/es.po]    new_index. Verifique que el disco no este lleno y la quota no ha"
level=warning msg="[po/es.po]    sido superada, y luego \"git restore --sateged :/\" para recuperar."
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: submodule--helper, subomdule--helper"
level=warning msg="[po/es.po]    >> msgid: submodule--helper print-default-remote takes no arguments"
level=warning msg="[po/es.po]    >> msgstr: subomdule--helper print-default-remote no toma argumentos"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: --raw, --stat"
level=warning msg="[po/es.po]    >> msgid: synonym for '-p --raw'"
level=warning msg="[po/es.po]    >> msgstr: sinónimo para '-p --stat'"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: --abort, s--abort"
level=warning msg="[po/es.po]    >> msgid: try \"git cherry-pick (--continue | %s--abort | --quit)\""
level=warning msg="[po/es.po]    >> msgstr: intenta \"git cherry-pick (--continue | --quit | %s --abort)\""
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: --abort, s--abort"
level=warning msg="[po/es.po]    >> msgid: try \"git revert (--continue | %s--abort | --quit)\""
level=warning msg="[po/es.po]    >> msgstr: intenta \"git revert (--continue | --quit | %s --abort)\""
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: load_cache_entires, load_cache_entries"
level=warning msg="[po/es.po]    >> msgid: unable to join load_cache_entries thread: %s"
level=warning msg="[po/es.po]    >> msgstr: no es posible unir hilo load_cache_entires: %s"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: %%(subject), %%(sujeto)"
level=warning msg="[po/es.po]    >> msgid: unrecognized %%(subject) argument: %s"
level=warning msg="[po/es.po]    >> msgstr: argumento %%(sujeto) no reconocido: %s"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: --reference, usa--reference"
level=warning msg="[po/es.po]    >> msgid: use --reference only while cloning"
level=warning msg="[po/es.po]    >> msgstr: usa--reference solamente si estás clonado"
level=warning msg="[po/es.po]"
level=warning msg="[po/es.po]    mismatched patterns: --group, --group=trailer"
level=warning msg="[po/es.po]    >> msgid: using --group=trailer with stdin is not supported"
level=warning msg="[po/es.po]    >> msgstr: el uso de --group = trailer con stdin no es compatible"
level=warning msg="[po/es.po]"
ERROR: check-po command failed
EOF

test_expect_success "check typos in es.po of git 2.31.1" '
	test_must_fail git -C workdir $HELPER check-po $POT_NO es >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=error msg="[po/es.po]    5210 translated messages."
level=error msg="[po/es.po]    too many obsolete entries (125) in comments, please remove them"
------------------------------------------------------------------------------
level=error msg="[po/es.po]    mismatched patterns: refs/remotes/<...>/HEAD, refs/remotos/<...>/HEAD"
level=error msg="[po/es.po]    >> msgid: delete refs/remotes/<name>/HEAD"
level=error msg="[po/es.po]    >> msgstr: borrar refs/remotos/<nombre>/HEAD"
level=error msg="[po/es.po]"
level=error msg="[po/es.po]    mismatched patterns: refs/preferch/, refs/prefetch/"
level=error msg="[po/es.po]    >> msgid: modify the refspec to place all refs within refs/prefetch/"
level=error msg="[po/es.po]    >> msgstr: modificar el refspec para colocar todas las referencias en refs/preferch/"
level=error msg="[po/es.po]"
ERROR: check-po command failed
EOF

test_expect_success "typos in master branch" '
	git -C workdir checkout master &&
	test_must_fail git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error es >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
