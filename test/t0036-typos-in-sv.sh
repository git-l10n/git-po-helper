#!/bin/sh

test_description="check typos in sv.po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn --report-file-locations=none"
POT_NO="--pot-file=no"

test_expect_success "checkout po-2.31.1" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1
'

cat >expect <<-\EOF
	ℹ️ Syntax check with msgfmt
	 INFO [sv.po] 5104 translated messages.
	❌ Obsolete #~ entries
	 ERROR [sv.po] you have 475 obsolete entries, please remove them
	⚠️ msgid/msgstr pattern check
	 WARNING [sv.po] mismatched patterns: --chmod, --chmod-parametern
	 WARNING [sv.po] >> msgid: --chmod param '%s' must be either -x or +x
	 WARNING [sv.po] >> msgstr: --chmod-parametern "%s" måste antingen vara -x eller +x
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --dump-alias, --dump-aliases
	 WARNING [sv.po] >> msgid: --dump-aliases incompatible with other options
	 WARNING [sv.po]
	 WARNING [sv.po] >> msgstr: --dump-alias är inkompatibelt med andra flaggor
	 WARNING [sv.po]
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --unpack-unreachable
	 WARNING [sv.po] >> msgid: --keep-unreachable and --unpack-unreachable are incompatible
	 WARNING [sv.po] >> msgstr: --keep-unreachable och -unpack-unreachable kan inte användas samtidigt
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --check
	 WARNING [sv.po] >> msgid: --name-only, --name-status, --check and -s are mutually exclusive
	 WARNING [sv.po] >> msgstr: --name-only, --name-status, -check och -s är ömsesidigt uteslutande
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --separate-git-dir, --separatebgit-dir
	 WARNING [sv.po] >> msgid: --separate-git-dir incompatible with bare repository
	 WARNING [sv.po] >> msgstr: --separatebgit-dir är inkompatibelt med naket arkiv
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --source=HEAD
	 WARNING [sv.po] >> msgid: Clone succeeded, but checkout failed.
	 WARNING [sv.po] You can inspect what was checked out with 'git status'
	 WARNING [sv.po] and retry with 'git restore --source=HEAD :/'
	 WARNING [sv.po]
	 WARNING [sv.po] >> msgstr: Klonen lyckades, men utcheckningen misslyckades.
	 WARNING [sv.po] Du kan inspektera det som checkades ut med "git status"
	 WARNING [sv.po] och försöka med "git restore -source=HEAD :/"
	 WARNING [sv.po]
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: refs/{heads,tags}/-prefix
	 WARNING [sv.po] >> msgid: The destination you provided is not a full refname (i.e.,
	 WARNING [sv.po] starting with "refs/"). We tried to guess what you meant by:
	 WARNING [sv.po]
	 WARNING [sv.po] - Looking for a ref that matches '%s' on the remote side.
	 WARNING [sv.po] - Checking if the <src> being pushed ('%s')
	 WARNING [sv.po] is a ref in "refs/{heads,tags}/". If so we add a corresponding
	 WARNING [sv.po] refs/{heads,tags}/ prefix on the remote side.
	 WARNING [sv.po]
	 WARNING [sv.po] Neither worked, so we gave up. You must fully qualify the ref.
	 WARNING [sv.po] >> msgstr: Målet du angav är inte ett komplett referensamn (dvs.,
	 WARNING [sv.po] startar med "refs/"). Vi försökte gissa vad du menade genom att:
	 WARNING [sv.po]
	 WARNING [sv.po] - Se efter en referens som motsvarar "%s" på fjärrsidan.
	 WARNING [sv.po] - Se om <källan> som sänds ("%s")
	 WARNING [sv.po] är en referens i "refs/{heads,tags}/". Om så lägger vi till
	 WARNING [sv.po] motsvarande refs/{heads,tags}/-prefix på fjärrsidan.
	 WARNING [sv.po]
	 WARNING [sv.po] Inget av dem fungerade, så vi gav upp. Ange fullständig referens.
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: add_cacheinfo, add_cahceinfo
	 WARNING [sv.po] >> msgid: add_cacheinfo failed for path '%s'; merge aborting.
	 WARNING [sv.po] >> msgstr: add_cahceinfo misslyckades för sökvägen "%s"; avslutar sammanslagningen.
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: dimmed_zebra
	 WARNING [sv.po] >> msgid: color moved setting must be one of 'no', 'default', 'blocks', 'zebra', 'dimmed-zebra', 'plain'
	 WARNING [sv.po] >> msgstr: färginställningen för flyttade block måste vara en av "no", "default", "blocks", "zebra", "dimmed_zebra", "plain"
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: %%(trailers:key=<...>), %%(trailers:nyckel=<...>)
	 WARNING [sv.po] >> msgid: expected %%(trailers:key=<value>)
	 WARNING [sv.po] >> msgstr: förvändate %%(trailers:nyckel=<värde>)
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: git bisect--helper, git-bisect--helper
	 WARNING [sv.po] >> msgid: git bisect--helper --bisect-terms [--term-good | --term-old | --term-bad | --term-new]
	 WARNING [sv.po] >> msgstr: git-bisect--helper --bisect-terms [--term-good | --term-old | --term-bad | --term-new]
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: git fetch-pack, git fetch-patch
	 WARNING [sv.po] >> msgid: git fetch-pack: fetch failed.
	 WARNING [sv.po] >> msgstr: git fetch-patch: hämtning misslyckades.
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: git-dir, git-dirs
	 WARNING [sv.po] >> msgid: git submodule--helper absorb-git-dirs [<options>] [<path>...]
	 WARNING [sv.po] >> msgstr: git submodule--helper absorb-git-dir [<flaggor>] [<sökväg>...]
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --branch, --brand
	 WARNING [sv.po] >> msgid: git submodule--helper set-branch [-q|--quiet] (-b|--branch) <branch> <path>
	 WARNING [sv.po] >> msgstr: git submodule--helper set-branch [-q|--quiet] (-b|--brand) <gren> <sökväg>
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --format=<...>...
	 WARNING [sv.po] >> msgid: git verify-tag [-v | --verbose] [--format=<format>] <tag>...
	 WARNING [sv.po] >> msgstr: git verify-tag [-v | --verbose] [--format=<format] <tagg>...
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: splitIndex.maxPercentChange, splitIndex.maxPercentage
	 WARNING [sv.po] >> msgid: splitIndex.maxPercentChange value '%d' should be between 0 and 100
	 WARNING [sv.po] >> msgstr: värdet "%d" för splitIndex.maxPercentage borde vara mellan 0 och 100
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: cache_entries, load_cache_entries
	 WARNING [sv.po] >> msgid: unable to create load_cache_entries thread: %s
	 WARNING [sv.po] >> msgstr: kunde inte läsa in cache_entries-tråden: %s
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: cache_entries, load_cache_entries
	 WARNING [sv.po] >> msgid: unable to join load_cache_entries thread: %s
	 WARNING [sv.po] >> msgstr: kunde inte utföra join på cache_entries-tråden: %s
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --stdin-commit, --stdin-commits
	 WARNING [sv.po] >> msgid: use at most one of --reachable, --stdin-commits, or --stdin-packs
	 WARNING [sv.po] >> msgstr: använd som mest en av --reachable, --stdin-commit och --stdin-packs
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --group, --group-flagga
	 WARNING [sv.po] >> msgid: using multiple --group options with stdin is not supported
	 WARNING [sv.po] >> msgstr: mer än en --group-flagga stöds inte med standard in
	 WARNING [sv.po]
	ERROR: check-po command failed
EOF

test_expect_success "check typos in sv.po" '
	test_must_fail git -C workdir $HELPER check-po $POT_NO \
		po/sv.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
	ℹ️ Syntax check with msgfmt
	 INFO [sv.po] 5104 translated messages.
	❌ Obsolete #~ entries
	 ERROR [sv.po] you have 475 obsolete entries, please remove them
	⚠️ msgid/msgstr pattern check
	 WARNING [sv.po] mismatched patterns: --chmod, --chmod-parametern
	 WARNING [sv.po] >> msgid: --chmod param '%s' must be either -x or +x
	 WARNING [sv.po] >> msgstr: --chmod-parametern "%s" måste antingen vara -x eller +x
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --dump-alias, --dump-aliases
	 WARNING [sv.po] >> msgid: --dump-aliases incompatible with other options
	 WARNING [sv.po]
	 WARNING [sv.po] >> msgstr: --dump-alias är inkompatibelt med andra flaggor
	 WARNING [sv.po]
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --unpack-unreachable
	 WARNING [sv.po] >> msgid: --keep-unreachable and --unpack-unreachable are incompatible
	 WARNING [sv.po] >> msgstr: --keep-unreachable och -unpack-unreachable kan inte användas samtidigt
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --check
	 WARNING [sv.po] >> msgid: --name-only, --name-status, --check and -s are mutually exclusive
	 WARNING [sv.po] >> msgstr: --name-only, --name-status, -check och -s är ömsesidigt uteslutande
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --separate-git-dir, --separatebgit-dir
	 WARNING [sv.po] >> msgid: --separate-git-dir incompatible with bare repository
	 WARNING [sv.po] >> msgstr: --separatebgit-dir är inkompatibelt med naket arkiv
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --source=HEAD
	 WARNING [sv.po] >> msgid: Clone succeeded, but checkout failed.
	 WARNING [sv.po] You can inspect what was checked out with 'git status'
	 WARNING [sv.po] and retry with 'git restore --source=HEAD :/'
	 WARNING [sv.po]
	 WARNING [sv.po] >> msgstr: Klonen lyckades, men utcheckningen misslyckades.
	 WARNING [sv.po] Du kan inspektera det som checkades ut med "git status"
	 WARNING [sv.po] och försöka med "git restore -source=HEAD :/"
	 WARNING [sv.po]
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: refs/{heads,tags}/-prefix
	 WARNING [sv.po] >> msgid: The destination you provided is not a full refname (i.e.,
	 WARNING [sv.po] starting with "refs/"). We tried to guess what you meant by:
	 WARNING [sv.po]
	 WARNING [sv.po] - Looking for a ref that matches '%s' on the remote side.
	 WARNING [sv.po] - Checking if the <src> being pushed ('%s')
	 WARNING [sv.po] is a ref in "refs/{heads,tags}/". If so we add a corresponding
	 WARNING [sv.po] refs/{heads,tags}/ prefix on the remote side.
	 WARNING [sv.po]
	 WARNING [sv.po] Neither worked, so we gave up. You must fully qualify the ref.
	 WARNING [sv.po] >> msgstr: Målet du angav är inte ett komplett referensamn (dvs.,
	 WARNING [sv.po] startar med "refs/"). Vi försökte gissa vad du menade genom att:
	 WARNING [sv.po]
	 WARNING [sv.po] - Se efter en referens som motsvarar "%s" på fjärrsidan.
	 WARNING [sv.po] - Se om <källan> som sänds ("%s")
	 WARNING [sv.po] är en referens i "refs/{heads,tags}/". Om så lägger vi till
	 WARNING [sv.po] motsvarande refs/{heads,tags}/-prefix på fjärrsidan.
	 WARNING [sv.po]
	 WARNING [sv.po] Inget av dem fungerade, så vi gav upp. Ange fullständig referens.
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: add_cacheinfo, add_cahceinfo
	 WARNING [sv.po] >> msgid: add_cacheinfo failed for path '%s'; merge aborting.
	 WARNING [sv.po] >> msgstr: add_cahceinfo misslyckades för sökvägen "%s"; avslutar sammanslagningen.
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: dimmed_zebra
	 WARNING [sv.po] >> msgid: color moved setting must be one of 'no', 'default', 'blocks', 'zebra', 'dimmed-zebra', 'plain'
	 WARNING [sv.po] >> msgstr: färginställningen för flyttade block måste vara en av "no", "default", "blocks", "zebra", "dimmed_zebra", "plain"
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: %%(trailers:key=<...>), %%(trailers:nyckel=<...>)
	 WARNING [sv.po] >> msgid: expected %%(trailers:key=<value>)
	 WARNING [sv.po] >> msgstr: förvändate %%(trailers:nyckel=<värde>)
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: git bisect--helper, git-bisect--helper
	 WARNING [sv.po] >> msgid: git bisect--helper --bisect-terms [--term-good | --term-old | --term-bad | --term-new]
	 WARNING [sv.po] >> msgstr: git-bisect--helper --bisect-terms [--term-good | --term-old | --term-bad | --term-new]
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: git fetch-pack, git fetch-patch
	 WARNING [sv.po] >> msgid: git fetch-pack: fetch failed.
	 WARNING [sv.po] >> msgstr: git fetch-patch: hämtning misslyckades.
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: git-dir, git-dirs
	 WARNING [sv.po] >> msgid: git submodule--helper absorb-git-dirs [<options>] [<path>...]
	 WARNING [sv.po] >> msgstr: git submodule--helper absorb-git-dir [<flaggor>] [<sökväg>...]
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --branch, --brand
	 WARNING [sv.po] >> msgid: git submodule--helper set-branch [-q|--quiet] (-b|--branch) <branch> <path>
	 WARNING [sv.po] >> msgstr: git submodule--helper set-branch [-q|--quiet] (-b|--brand) <gren> <sökväg>
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --format=<...>...
	 WARNING [sv.po] >> msgid: git verify-tag [-v | --verbose] [--format=<format>] <tag>...
	 WARNING [sv.po] >> msgstr: git verify-tag [-v | --verbose] [--format=<format] <tagg>...
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: splitIndex.maxPercentChange, splitIndex.maxPercentage
	 WARNING [sv.po] >> msgid: splitIndex.maxPercentChange value '%d' should be between 0 and 100
	 WARNING [sv.po] >> msgstr: värdet "%d" för splitIndex.maxPercentage borde vara mellan 0 och 100
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: cache_entries, load_cache_entries
	 WARNING [sv.po] >> msgid: unable to create load_cache_entries thread: %s
	 WARNING [sv.po] >> msgstr: kunde inte läsa in cache_entries-tråden: %s
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: cache_entries, load_cache_entries
	 WARNING [sv.po] >> msgid: unable to join load_cache_entries thread: %s
	 WARNING [sv.po] >> msgstr: kunde inte utföra join på cache_entries-tråden: %s
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --stdin-commit, --stdin-commits
	 WARNING [sv.po] >> msgid: use at most one of --reachable, --stdin-commits, or --stdin-packs
	 WARNING [sv.po] >> msgstr: använd som mest en av --reachable, --stdin-commit och --stdin-packs
	 WARNING [sv.po]
	 WARNING [sv.po] mismatched patterns: --group, --group-flagga
	 WARNING [sv.po] >> msgid: using multiple --group options with stdin is not supported
	 WARNING [sv.po] >> msgstr: mer än en --group-flagga stöds inte med standard in
	 WARNING [sv.po]
	ERROR: check-po command failed
EOF

test_expect_success "typos in master branch" '
	git -C workdir checkout master &&
	test_must_fail git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error \
		po/sv.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cat >expect <<-\EOF &&
	ℹ️ Syntax check with msgfmt
	 INFO [sv.po] 5282 translated messages.
	❌ Obsolete #~ entries
	 ERROR [sv.po] you have 768 obsolete entries, please remove them
	❌ msgid/msgstr pattern check
	 ERROR [sv.po] mismatched patterns: refs/{heads,tags}/-prefix
	 ERROR [sv.po] >> msgid: The destination you provided is not a full refname (i.e.,
	 ERROR [sv.po] starting with "refs/"). We tried to guess what you meant by:
	 ERROR [sv.po]
	 ERROR [sv.po] - Looking for a ref that matches '\''%s'\'' on the remote side.
	 ERROR [sv.po] - Checking if the <src> being pushed ('\''%s'\'')
	 ERROR [sv.po] is a ref in "refs/{heads,tags}/". If so we add a corresponding
	 ERROR [sv.po] refs/{heads,tags}/ prefix on the remote side.
	 ERROR [sv.po]
	 ERROR [sv.po] Neither worked, so we gave up. You must fully qualify the ref.
	 ERROR [sv.po] >> msgstr: Målet du angav är inte ett komplett referensamn (dvs.,
	 ERROR [sv.po] startar med "refs/"). Vi försökte gissa vad du menade genom att:
	 ERROR [sv.po]
	 ERROR [sv.po] - Se efter en referens som motsvarar "%s" på fjärrsidan.
	 ERROR [sv.po] - Se om <källan> som sänds ("%s")
	 ERROR [sv.po] är en referens i "refs/{heads,tags}/". Om så lägger vi till
	 ERROR [sv.po] motsvarande refs/{heads,tags}/-prefix på fjärrsidan.
	 ERROR [sv.po]
	 ERROR [sv.po] Inget av dem fungerade, så vi gav upp. Ange fullständig referens.
	 ERROR [sv.po]
	ERROR: check-po command failed
	EOF
	test_cmp expect actual
'

test_done
