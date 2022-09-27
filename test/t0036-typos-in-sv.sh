#!/bin/sh

test_description="check typos in sv.po"

. ./lib/sharness.sh

HELPER="po-helper --no-special-gettext-versions --pot-file=no --report-typos=warn --report-file-locations=none"

test_expect_success "checkout po-2.31.1" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=error msg="[po/sv.po]    5104 translated messages."
level=error msg="[po/sv.po]    too many obsolete entries (475) in comments, please remove them"
------------------------------------------------------------------------------
level=warning msg="[po/sv.po]    mismatch variable names: --chmod, --chmod-parametern"
level=warning msg="[po/sv.po]    >> msgid: --chmod param '%s' must be either -x or +x"
level=warning msg="[po/sv.po]    >> msgstr: --chmod-parametern \"%s\" måste antingen vara -x eller +x"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: --dump-alias, --dump-aliases"
level=warning msg="[po/sv.po]    >> msgid: --dump-aliases incompatible with other options"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    >> msgstr: --dump-alias är inkompatibelt med andra flaggor"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: --unpack-unreachable"
level=warning msg="[po/sv.po]    >> msgid: --keep-unreachable and --unpack-unreachable are incompatible"
level=warning msg="[po/sv.po]    >> msgstr: --keep-unreachable och -unpack-unreachable kan inte användas samtidigt"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: --check"
level=warning msg="[po/sv.po]    >> msgid: --name-only, --name-status, --check and -s are mutually exclusive"
level=warning msg="[po/sv.po]    >> msgstr: --name-only, --name-status, -check och -s är ömsesidigt uteslutande"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: --separate-git-dir, --separatebgit-dir"
level=warning msg="[po/sv.po]    >> msgid: --separate-git-dir incompatible with bare repository"
level=warning msg="[po/sv.po]    >> msgstr: --separatebgit-dir är inkompatibelt med naket arkiv"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: --source=HEAD"
level=warning msg="[po/sv.po]    >> msgid: Clone succeeded, but checkout failed."
level=warning msg="[po/sv.po]    You can inspect what was checked out with 'git status'"
level=warning msg="[po/sv.po]    and retry with 'git restore --source=HEAD :/'"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    >> msgstr: Klonen lyckades, men utcheckningen misslyckades."
level=warning msg="[po/sv.po]    Du kan inspektera det som checkades ut med \"git status\""
level=warning msg="[po/sv.po]    och försöka med \"git restore -source=HEAD :/\""
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: add_cacheinfo, add_cahceinfo"
level=warning msg="[po/sv.po]    >> msgid: add_cacheinfo failed for path '%s'; merge aborting."
level=warning msg="[po/sv.po]    >> msgstr: add_cahceinfo misslyckades för sökvägen \"%s\"; avslutar sammanslagningen."
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: dimmed_zebra"
level=warning msg="[po/sv.po]    >> msgid: color moved setting must be one of 'no', 'default', 'blocks', 'zebra', 'dimmed-zebra', 'plain'"
level=warning msg="[po/sv.po]    >> msgstr: färginställningen för flyttade block måste vara en av \"no\", \"default\", \"blocks\", \"zebra\", \"dimmed_zebra\", \"plain\""
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: %%(trailers:key=<...>), %%(trailers:nyckel=<...>)"
level=warning msg="[po/sv.po]    >> msgid: expected %%(trailers:key=<value>)"
level=warning msg="[po/sv.po]    >> msgstr: förvändate %%(trailers:nyckel=<värde>)"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: git bisect--helper, git-bisect--helper"
level=warning msg="[po/sv.po]    >> msgid: git bisect--helper --bisect-terms [--term-good | --term-old | --term-bad | --term-new]"
level=warning msg="[po/sv.po]    >> msgstr: git-bisect--helper --bisect-terms [--term-good | --term-old | --term-bad | --term-new]"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: git fetch-pack, git fetch-patch"
level=warning msg="[po/sv.po]    >> msgid: git fetch-pack: fetch failed."
level=warning msg="[po/sv.po]    >> msgstr: git fetch-patch: hämtning misslyckades."
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: git-dir, git-dirs"
level=warning msg="[po/sv.po]    >> msgid: git submodule--helper absorb-git-dirs [<options>] [<path>...]"
level=warning msg="[po/sv.po]    >> msgstr: git submodule--helper absorb-git-dir [<flaggor>] [<sökväg>...]"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: --branch, --brand"
level=warning msg="[po/sv.po]    >> msgid: git submodule--helper set-branch [-q|--quiet] (-b|--branch) <branch> <path>"
level=warning msg="[po/sv.po]    >> msgstr: git submodule--helper set-branch [-q|--quiet] (-b|--brand) <gren> <sökväg>"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: --format="
level=warning msg="[po/sv.po]    >> msgid: git verify-tag [-v | --verbose] [--format=<format>] <tag>..."
level=warning msg="[po/sv.po]    >> msgstr: git verify-tag [-v | --verbose] [--format=<format] <tagg>..."
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: splitIndex.maxPercentChange, splitIndex.maxPercentage"
level=warning msg="[po/sv.po]    >> msgid: splitIndex.maxPercentChange value '%d' should be between 0 and 100"
level=warning msg="[po/sv.po]    >> msgstr: värdet \"%d\" för splitIndex.maxPercentage borde vara mellan 0 och 100"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: cache_entries, load_cache_entries"
level=warning msg="[po/sv.po]    >> msgid: unable to create load_cache_entries thread: %s"
level=warning msg="[po/sv.po]    >> msgstr: kunde inte läsa in cache_entries-tråden: %s"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: cache_entries, load_cache_entries"
level=warning msg="[po/sv.po]    >> msgid: unable to join load_cache_entries thread: %s"
level=warning msg="[po/sv.po]    >> msgstr: kunde inte utföra join på cache_entries-tråden: %s"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: --stdin-commit, --stdin-commits"
level=warning msg="[po/sv.po]    >> msgid: use at most one of --reachable, --stdin-commits, or --stdin-packs"
level=warning msg="[po/sv.po]    >> msgstr: använd som mest en av --reachable, --stdin-commit och --stdin-packs"
level=warning msg="[po/sv.po]"
level=warning msg="[po/sv.po]    mismatch variable names: --group, --group-flagga"
level=warning msg="[po/sv.po]    >> msgid: using multiple --group options with stdin is not supported"
level=warning msg="[po/sv.po]    >> msgstr: mer än en --group-flagga stöds inte med standard in"
level=warning msg="[po/sv.po]"

ERROR: fail to execute "git-po-helper check-po"
EOF

test_expect_success "check typos in sv.po" '
	test_must_fail git -C workdir $HELPER check-po sv >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=error msg="[po/sv.po]    5282 translated messages."
level=error msg="[po/sv.po]    too many obsolete entries (768) in comments, please remove them"

ERROR: fail to execute "git-po-helper check-po"
EOF

test_expect_success "no typos in master branch" '
	git -C workdir checkout master &&
	test_must_fail git -C workdir $HELPER \
		check-po --report-typos=error sv >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
