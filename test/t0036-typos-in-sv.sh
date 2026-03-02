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
------------------------------------------------------------------------------
level=error msg="[po/sv.po]    5104 translated messages."
level=error msg="[po/sv.po]    too many obsolete entries (475) in comments, please remove them"
------------------------------------------------------------------------------
level=warning msg="[po/sv.po]    mismatched patterns: --chmod, --chmod-parametern"
level=warning msg="[po/sv.po]    >> msgid: --chmod param '%s' must be either -x or +x"
level=warning msg="[po/sv.po]    >> msgstr: --chmod-parametern \"%s\" måste antingen vara -x eller +x"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: --dump-alias, --dump-aliases"
level=warning msg="[po/sv.po]    >> msgid: --dump-aliases incompatible with other options"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    >> msgstr: --dump-alias är inkompatibelt med andra flaggor"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: --unpack-unreachable"
level=warning msg="[po/sv.po]    >> msgid: --keep-unreachable and --unpack-unreachable are incompatible"
level=warning msg="[po/sv.po]    >> msgstr: --keep-unreachable och -unpack-unreachable kan inte användas samtidigt"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: --check"
level=warning msg="[po/sv.po]    >> msgid: --name-only, --name-status, --check and -s are mutually exclusive"
level=warning msg="[po/sv.po]    >> msgstr: --name-only, --name-status, -check och -s är ömsesidigt uteslutande"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: --separate-git-dir, --separatebgit-dir"
level=warning msg="[po/sv.po]    >> msgid: --separate-git-dir incompatible with bare repository"
level=warning msg="[po/sv.po]    >> msgstr: --separatebgit-dir är inkompatibelt med naket arkiv"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: --source=HEAD"
level=warning msg="[po/sv.po]    >> msgid: Clone succeeded, but checkout failed."
level=warning msg="[po/sv.po]    You can inspect what was checked out with 'git status'"
level=warning msg="[po/sv.po]    and retry with 'git restore --source=HEAD :/'"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    >> msgstr: Klonen lyckades, men utcheckningen misslyckades."
level=warning msg="[po/sv.po]    Du kan inspektera det som checkades ut med \"git status\""
level=warning msg="[po/sv.po]    och försöka med \"git restore -source=HEAD :/\""
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: refs/{heads,tags}/-prefix"
level=warning msg="[po/sv.po]    >> msgid: The destination you provided is not a full refname (i.e.,"
level=warning msg="[po/sv.po]    starting with \"refs/\"). We tried to guess what you meant by:"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    - Looking for a ref that matches '%s' on the remote side."
level=warning msg="[po/sv.po]    - Checking if the <src> being pushed ('%s')"
level=warning msg="[po/sv.po]     is a ref in \"refs/{heads,tags}/\". If so we add a corresponding"
level=warning msg="[po/sv.po]     refs/{heads,tags}/ prefix on the remote side."
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    Neither worked, so we gave up. You must fully qualify the ref."
level=warning msg="[po/sv.po]    >> msgstr: Målet du angav är inte ett komplett referensamn (dvs.,"
level=warning msg="[po/sv.po]    startar med \"refs/\"). Vi försökte gissa vad du menade genom att:"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    - Se efter en referens som motsvarar \"%s\" på fjärrsidan."
level=warning msg="[po/sv.po]    - Se om <källan> som sänds (\"%s\")"
level=warning msg="[po/sv.po]     är en referens i \"refs/{heads,tags}/\". Om så lägger vi till"
level=warning msg="[po/sv.po]     motsvarande refs/{heads,tags}/-prefix på fjärrsidan."
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    Inget av dem fungerade, så vi gav upp. Ange fullständig referens."
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: add_cacheinfo, add_cahceinfo"
level=warning msg="[po/sv.po]    >> msgid: add_cacheinfo failed for path '%s'; merge aborting."
level=warning msg="[po/sv.po]    >> msgstr: add_cahceinfo misslyckades för sökvägen \"%s\"; avslutar sammanslagningen."
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: dimmed_zebra"
level=warning msg="[po/sv.po]    >> msgid: color moved setting must be one of 'no', 'default', 'blocks', 'zebra', 'dimmed-zebra', 'plain'"
level=warning msg="[po/sv.po]    >> msgstr: färginställningen för flyttade block måste vara en av \"no\", \"default\", \"blocks\", \"zebra\", \"dimmed_zebra\", \"plain\""
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: %%(trailers:key=<...>), %%(trailers:nyckel=<...>)"
level=warning msg="[po/sv.po]    >> msgid: expected %%(trailers:key=<value>)"
level=warning msg="[po/sv.po]    >> msgstr: förvändate %%(trailers:nyckel=<värde>)"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: git bisect--helper, git-bisect--helper"
level=warning msg="[po/sv.po]    >> msgid: git bisect--helper --bisect-terms [--term-good | --term-old | --term-bad | --term-new]"
level=warning msg="[po/sv.po]    >> msgstr: git-bisect--helper --bisect-terms [--term-good | --term-old | --term-bad | --term-new]"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: git fetch-pack, git fetch-patch"
level=warning msg="[po/sv.po]    >> msgid: git fetch-pack: fetch failed."
level=warning msg="[po/sv.po]    >> msgstr: git fetch-patch: hämtning misslyckades."
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: git-dir, git-dirs"
level=warning msg="[po/sv.po]    >> msgid: git submodule--helper absorb-git-dirs [<options>] [<path>...]"
level=warning msg="[po/sv.po]    >> msgstr: git submodule--helper absorb-git-dir [<flaggor>] [<sökväg>...]"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: --branch, --brand"
level=warning msg="[po/sv.po]    >> msgid: git submodule--helper set-branch [-q|--quiet] (-b|--branch) <branch> <path>"
level=warning msg="[po/sv.po]    >> msgstr: git submodule--helper set-branch [-q|--quiet] (-b|--brand) <gren> <sökväg>"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: --format=<...>..."
level=warning msg="[po/sv.po]    >> msgid: git verify-tag [-v | --verbose] [--format=<format>] <tag>..."
level=warning msg="[po/sv.po]    >> msgstr: git verify-tag [-v | --verbose] [--format=<format] <tagg>..."
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: splitIndex.maxPercentChange, splitIndex.maxPercentage"
level=warning msg="[po/sv.po]    >> msgid: splitIndex.maxPercentChange value '%d' should be between 0 and 100"
level=warning msg="[po/sv.po]    >> msgstr: värdet \"%d\" för splitIndex.maxPercentage borde vara mellan 0 och 100"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: cache_entries, load_cache_entries"
level=warning msg="[po/sv.po]    >> msgid: unable to create load_cache_entries thread: %s"
level=warning msg="[po/sv.po]    >> msgstr: kunde inte läsa in cache_entries-tråden: %s"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: cache_entries, load_cache_entries"
level=warning msg="[po/sv.po]    >> msgid: unable to join load_cache_entries thread: %s"
level=warning msg="[po/sv.po]    >> msgstr: kunde inte utföra join på cache_entries-tråden: %s"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: --stdin-commit, --stdin-commits"
level=warning msg="[po/sv.po]    >> msgid: use at most one of --reachable, --stdin-commits, or --stdin-packs"
level=warning msg="[po/sv.po]    >> msgstr: använd som mest en av --reachable, --stdin-commit och --stdin-packs"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatched patterns: --group, --group-flagga"
level=warning msg="[po/sv.po]    >> msgid: using multiple --group options with stdin is not supported"
level=warning msg="[po/sv.po]    >> msgstr: mer än en --group-flagga stöds inte med standard in"
level=warning msg="[po/sv.po]"
ERROR: check-po command failed
EOF

test_expect_success "check typos in sv.po" '
	test_must_fail git -C workdir $HELPER check-po $POT_NO sv >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=error msg="[po/sv.po]    5282 translated messages."
level=error msg="[po/sv.po]    too many obsolete entries (768) in comments, please remove them"
------------------------------------------------------------------------------
level=error msg="[po/sv.po]    mismatched patterns: refs/{heads,tags}/-prefix"
level=error msg="[po/sv.po]    >> msgid: The destination you provided is not a full refname (i.e.,"
level=error msg="[po/sv.po]    starting with \"refs/\"). We tried to guess what you meant by:"
level=error msg="[po/sv.po]"
level=error msg="[po/sv.po]    - Looking for a ref that matches '%s' on the remote side."
level=error msg="[po/sv.po]    - Checking if the <src> being pushed ('%s')"
level=error msg="[po/sv.po]     is a ref in \"refs/{heads,tags}/\". If so we add a corresponding"
level=error msg="[po/sv.po]     refs/{heads,tags}/ prefix on the remote side."
level=error msg="[po/sv.po]"
level=error msg="[po/sv.po]    Neither worked, so we gave up. You must fully qualify the ref."
level=error msg="[po/sv.po]    >> msgstr: Målet du angav är inte ett komplett referensamn (dvs.,"
level=error msg="[po/sv.po]    startar med \"refs/\"). Vi försökte gissa vad du menade genom att:"
level=error msg="[po/sv.po]"
level=error msg="[po/sv.po]    - Se efter en referens som motsvarar \"%s\" på fjärrsidan."
level=error msg="[po/sv.po]    - Se om <källan> som sänds (\"%s\")"
level=error msg="[po/sv.po]     är en referens i \"refs/{heads,tags}/\". Om så lägger vi till"
level=error msg="[po/sv.po]     motsvarande refs/{heads,tags}/-prefix på fjärrsidan."
level=error msg="[po/sv.po]"
level=error msg="[po/sv.po]    Inget av dem fungerade, så vi gav upp. Ange fullständig referens."
level=error msg="[po/sv.po]"
ERROR: check-po command failed
EOF

test_expect_success "typos in master branch" '
	git -C workdir checkout master &&
	test_must_fail git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error sv >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
