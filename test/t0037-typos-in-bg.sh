#!/bin/sh

test_description="check typos in bg.po"

. ./lib/sharness.sh

HELPER="po-helper --no-gettext-back-compatible"

test_expect_success "checkout po-2.31.1" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1
'

cat >expect <<-\EOF
level=info msg="[po/bg.po]    5104 translated messages."
level=warning msg="[po/bg.po]    mismatch variable names: example.com"
level=warning msg="[po/bg.po]    >> msgid: \n*** Please tell me who you are.\n\nRun\n\n git config --global user.email \"you@example.com\"\n git config --global user.name \"Your Name\"\n\nto set your account's default identity.\nOmit --global to set the identity only in this repository.\n\n"
level=warning msg="[po/bg.po]    >> msgstr: \n●●● Въведете самоличност.\n\nИзпълнете:\n\n git config --global user.email \"ИМЕ@example.com\"\n git config --global user.name \"ВАШЕТО ИМЕ\"\n\nи въведете данни за себе си.\nАко пропуснете опцията „--global“, самоличността е само за текущото хранилище.\n\n"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: $HOME"
level=warning msg="[po/bg.po]    >> msgid: $HOME not set"
level=warning msg="[po/bg.po]    >> msgstr: променливата „HOME“ не е зададена"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --git-dir=, --work-tree="
level=warning msg="[po/bg.po]    >> msgid: %s (or --work-tree=<directory>) not allowed without specifying %s (or --git-dir=<directory>)"
level=warning msg="[po/bg.po]    >> msgstr: %s (или --work-tree=ДИРЕКТОРИЯ) изисква указването на %s (или --git-dir=ДИРЕКТОРИЯ)"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: new_index"
level=warning msg="[po/bg.po]    >> msgid: %s: Unable to write new index file"
level=warning msg="[po/bg.po]    >> msgstr: %s: новият индекс не може да бъде запазен"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --abort"
level=warning msg="[po/bg.po]    >> msgid: --abort but leave index and working tree alone"
level=warning msg="[po/bg.po]    >> msgstr: преустановяване без промяна на индекса и работното дърво"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --contains"
level=warning msg="[po/bg.po]    >> msgid: --contains option is only allowed in list mode"
level=warning msg="[po/bg.po]    >> msgstr: Опцията „-contains“ изисква режим на списък."
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: extensions.partialClone, extensions.partialclone"
level=warning msg="[po/bg.po]    >> msgid: --filter can only be used with the remote configured in extensions.partialclone"
level=warning msg="[po/bg.po]    >> msgstr: опцията „--filter“ може да се ползва само с отдалеченото хранилище указано в настройката „extensions.partialClone“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --no-contains"
level=warning msg="[po/bg.po]    >> msgid: --no-contains option is only allowed in list mode"
level=warning msg="[po/bg.po]    >> msgstr: Опцията „-contains“ изисква режим на списък."
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --cached, --no-index"
level=warning msg="[po/bg.po]    >> msgid: --no-index or --untracked cannot be used with revs"
level=warning msg="[po/bg.po]    >> msgstr: опциите „--cached“ и „--untracked“ са несъвместими с версии."
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --hard, --mixed, --soft"
level=warning msg="[po/bg.po]    >> msgid: --patch is incompatible with --{hard,mixed,soft}"
level=warning msg="[po/bg.po]    >> msgstr: опцията „--patch“ е несъвместима с всяка от опциите „--hard“/„--mixed“/„--soft“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --points-at"
level=warning msg="[po/bg.po]    >> msgid: --points-at option is only allowed in list mode"
level=warning msg="[po/bg.po]    >> msgstr: Опцията „-points-at“ изисква режим на списък."
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --reflog, --track"
level=warning msg="[po/bg.po]    >> msgid: --reflog option needs one branch name"
level=warning msg="[po/bg.po]    >> msgstr: опцията „--track“ изисква точно едно име на клон"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --stdin"
level=warning msg="[po/bg.po]    >> msgid: --stdin and --merge-base are mutually exclusive"
level=warning msg="[po/bg.po]    >> msgstr: опциите „-stdin“ и „--merge-base“ са несъвместими"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --rfc"
level=warning msg="[po/bg.po]    >> msgid: --subject-prefix/--rfc and -k are mutually exclusive"
level=warning msg="[po/bg.po]    >> msgstr: опциите „--subject-prefix“/„-rfc“ и „-k“ са несъвместими"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --name-only, --only-input"
level=warning msg="[po/bg.po]    >> msgid: --trailer with --only-input does not make sense"
level=warning msg="[po/bg.po]    >> msgstr: опцията „--trailer“ е несъвместима с „--name-only“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --unpacked="
level=warning msg="[po/bg.po]    >> msgid: --unpacked=<packfile> no longer supported"
level=warning msg="[po/bg.po]    >> msgstr: опцията „--unpacked=ПАКЕТЕН_ФАЙЛ“ вече не се поддържа"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --worktre, --worktree"
level=warning msg="[po/bg.po]    >> msgid: --worktree cannot be used with multiple working trees unless the config\nextension worktreeConfig is enabled. Please read \"CONFIGURATION FILE\"\nsection in \"git help worktree\" for details"
level=warning msg="[po/bg.po]    >> msgstr: опцията „--worktre“ не приема множество работни дървета, преди\nвключването на разширението в настройките „worktreeConfig“. За\nповече информация вижте раздела „CONFIGURATION FILE“ в\n„git help worktree“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --no-commit"
level=warning msg="[po/bg.po]    >> msgid: Automatic merge went well; stopped before committing as requested\n"
level=warning msg="[po/bg.po]    >> msgstr: Автоматичното сливане завърши успешно. Самото подаване не е извършено, защото бе зададена опцията „--no-commit“.\n"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: git pack-objects"
level=warning msg="[po/bg.po]    >> msgid: Could not spawn pack-objects"
level=warning msg="[po/bg.po]    >> msgstr: Командата „git pack-objects“ не може да бъде стартирана"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: $BRANCH, $br"
level=warning msg="[po/bg.po]    >> msgid: Git normally never creates a ref that ends with 40 hex characters\nbecause it will be ignored when you just specify 40-hex. These refs\nmay be created by mistake. For example,\n\n git switch -c $br $(git rev-parse ...)\n\nwhere \"$br\" is somehow empty and a 40-hex ref is created. Please\nexamine these refs and maybe delete them. Turn this message off by\nrunning \"git config advice.objectNameWarning false\""
level=warning msg="[po/bg.po]    >> msgstr: При нормална работа Git никога не създава указатели, които завършват\nс 40 шестнадесетични знака, защото стандартно те ще бъдат прескачани.\nВъзможно е такива указатели да са създадени случайно. Например:\n\n git switch -c $BRANCH $(git rev-parse …)\n\nкъдето стойността на променливата на средата BRANCH е празна, при което\nсе създава подобен указател. Прегледайте тези указатели и ги изтрийте.\nЗа да изключите това съобщение, изпълнете:\n\n git config advice.objectNameWarning false"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: git-am"
level=warning msg="[po/bg.po]    >> msgid: It looks like 'git am' is in progress. Cannot rebase."
level=warning msg="[po/bg.po]    >> msgstr: Изглежда, че сега се прилагат кръпки чрез командата „git-am“. Не може да пребазирате в момента."
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --no-ff"
level=warning msg="[po/bg.po]    >> msgid: Non-fast-forward commit does not make sense into an empty head"
level=warning msg="[po/bg.po]    >> msgstr: Понеже върхът е без история, всички сливания са превъртания, не може да се извърши същинско сливане изисквано от опцията „--no-ff“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: example.com"
level=warning msg="[po/bg.po]    >> msgid: Your name and email address were configured automatically based\non your username and hostname. Please check that they are accurate.\nYou can suppress this message by setting them explicitly:\n\n git config --global user.name \"Your Name\"\n git config --global user.email you@example.com\n\nAfter doing this, you may fix the identity used for this commit with:\n\n git commit --amend --reset-author\n"
level=warning msg="[po/bg.po]    >> msgstr: Името и адресът за е-поща са настроени автоматично на базата на името на\nпотребителя и името на машината. Проверете дали са верни. Можете да спрете\nтова съобщение като изрично зададете стойностите:\n\n git config --global user.name \"Вашето Име\"\n git config --global user.email пенчо@example.com\n\nСлед като направите това, можете да коригирате информацията за автора на\nтекущото подаване чрез:\n\n git commit --amend --reset-author\n"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --update"
level=warning msg="[po/bg.po]    >> msgid: bad value for update parameter"
level=warning msg="[po/bg.po]    >> msgstr: неправилен параметър към опцията „--update“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --filter, --stdout"
level=warning msg="[po/bg.po]    >> msgid: cannot use --filter without --stdout"
level=warning msg="[po/bg.po]    >> msgstr: опцията „-filter“ изисква „-stdout“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: dimmed_zebra"
level=warning msg="[po/bg.po]    >> msgid: color moved setting must be one of 'no', 'default', 'blocks', 'zebra', 'dimmed-zebra', 'plain'"
level=warning msg="[po/bg.po]    >> msgstr: настройката за цвят за преместване трябва да е една от: „no“ (без), „default“ (стандартно), „blocks“ (парчета), „zebra“ (райе), „dimmed_zebra“ (тъмно райе), „plain“ (обикновено)"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: conclude_pack"
level=warning msg="[po/bg.po]    >> msgid: confusion beyond insanity"
level=warning msg="[po/bg.po]    >> msgstr: фатална грешка във функцията „conclude_pack“. Това е грешка в Git, докладвайте я на разработчиците, като пратите е-писмо на адрес: „git@vger.kernel.org“."
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --bare"
level=warning msg="[po/bg.po]    >> msgid: create a mirror repository (implies bare)"
level=warning msg="[po/bg.po]    >> msgstr: създаване на хранилище-огледало (включва опцията „--bare“ за голо хранилище)"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: credential-cache--daemon"
level=warning msg="[po/bg.po]    >> msgid: credential-cache--daemon unavailable; no unix socket support"
level=warning msg="[po/bg.po]    >> msgstr: демонът за кеша с идентификациите е недостъпен — липсва поддръжка на гнезда на unix"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: git-difftool"
level=warning msg="[po/bg.po]    >> msgid: difftool requires worktree or --no-index"
level=warning msg="[po/bg.po]    >> msgstr: „git-difftool“ изисква работно дърво или опцията „--no-index“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --word-diff-regex="
level=warning msg="[po/bg.po]    >> msgid: equivalent to --word-diff=color --word-diff-regex=<regex>"
level=warning msg="[po/bg.po]    >> msgstr: псевдоним на „--word-diff=color --word-diff-regex=РЕГУЛЯРЕН_ИЗРАЗ“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: %%(trailers:key=<...>), %%(trailers:КЛЮЧ=СТОЙНОСТ)"
level=warning msg="[po/bg.po]    >> msgid: expected %%(trailers:key=<value>)"
level=warning msg="[po/bg.po]    >> msgstr: очаква се %%(trailers:КЛЮЧ=СТОЙНОСТ)"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: %%(align:<...>,<...>), %%(align:ШИРОЧИНА,ПОЗИЦИЯ)"
level=warning msg="[po/bg.po]    >> msgid: expected format: %%(align:<width>,<position>)"
level=warning msg="[po/bg.po]    >> msgstr: очакван формат: %%(align:ШИРОЧИНА,ПОЗИЦИЯ)"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: %%(color:<...>), %%(color:ЦВЯТ)"
level=warning msg="[po/bg.po]    >> msgid: expected format: %%(color:<color>)"
level=warning msg="[po/bg.po]    >> msgstr: очакван формат: %%(color:ЦВЯТ)"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --quit, --skip"
level=warning msg="[po/bg.po]    >> msgid: git am [<options>] (--continue | --skip | --abort)"
level=warning msg="[po/bg.po]    >> msgstr: git am [ОПЦИЯ…] (--continue | --quit | --abort)"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --bisect-next, --bisect-replay"
level=warning msg="[po/bg.po]    >> msgid: git bisect--helper --bisect-replay <filename>"
level=warning msg="[po/bg.po]    >> msgstr: git bisect--helper --bisect-next ИМЕ_НА_ФАЙЛ"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --bisect-reset, --bisect-skip"
level=warning msg="[po/bg.po]    >> msgid: git bisect--helper --bisect-skip [(<rev>|<range>)...]"
level=warning msg="[po/bg.po]    >> msgstr: git bisect--helper --bisect-reset [(ВЕРСИЯ|ДИАПАЗОН)…]"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --bisect-reset, --bisect-state"
level=warning msg="[po/bg.po]    >> msgid: git bisect--helper --bisect-state (bad|new) [<rev>]"
level=warning msg="[po/bg.po]    >> msgstr: git bisect--helper --bisect-reset (ЛОШО) [ВЕРСИЯ]"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --bisect-reset, --bisect-state"
level=warning msg="[po/bg.po]    >> msgid: git bisect--helper --bisect-state (good|old) [<rev>...]"
level=warning msg="[po/bg.po]    >> msgstr: git bisect--helper --bisect-reset (ДОБРО) [ВЕРСИЯ…]"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --config="
level=warning msg="[po/bg.po]    >> msgid: git for-each-repo --config=<config> <command-args>"
level=warning msg="[po/bg.po]    >> msgstr: git for-each-repo --config=НАСТРОЙКА АРГУМЕНТ…"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --batch-size="
level=warning msg="[po/bg.po]    >> msgid: git multi-pack-index [<options>] (write|verify|expire|repack --batch-size=<size>)"
level=warning msg="[po/bg.po]    >> msgstr: git multi-pack-index [ОПЦИЯ…] (write|verify|expire|repack --batch-size=РАЗМЕР)"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --prefix="
level=warning msg="[po/bg.po]    >> msgid: git read-tree [(-m [--trivial] [--aggressive] | --reset | --prefix=<prefix>) [-u [--exclude-per-directory=<gitignore>] | -i]] [--no-sparse-checkout] [--index-output=<file>] (--empty | <tree-ish1> [<tree-ish2> [<tree-ish3>]])"
level=warning msg="[po/bg.po]    >> msgstr: git read-tree [(-m [--trivial] [--aggressive] | --reset | --prefix=ПРЕФИКС) [-u [--exclude-per-directory=ФАЙЛ_С_ИЗКЛЮЧЕНИЯ] | -i]] [--no-sparse-checkout] [--index-output=ФАЙЛ] (--empty | УКАЗАТЕЛ_КЪМ_ДЪРВО_1 [УКАЗАТЕЛ_КЪМ_ДЪРВО_2 [УКАЗАТЕЛ_КЪМ_ДЪРВО_3]])"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: git upload-pack, git upload-repack"
level=warning msg="[po/bg.po]    >> msgid: git upload-pack [<options>] <dir>"
level=warning msg="[po/bg.po]    >> msgstr: git upload-repack [ОПЦИЯ…] ДИРЕКТОРИЯ"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: git verify-commit, git verify-tag"
level=warning msg="[po/bg.po]    >> msgid: git verify-commit [-v | --verbose] <commit>..."
level=warning msg="[po/bg.po]    >> msgstr: git verify-tag [-v | --verbose] ПОДАВАНЕ…"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --force"
level=warning msg="[po/bg.po]    >> msgid: helper %s does not support 'force'"
level=warning msg="[po/bg.po]    >> msgstr: насрещната помощна програма не поддържа „%s“ поддържа опцията „--force“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: crlf_action"
level=warning msg="[po/bg.po]    >> msgid: illegal crlf_action %d"
level=warning msg="[po/bg.po]    >> msgstr: неправилно действие за край на ред: %d"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: git index-pack"
level=warning msg="[po/bg.po]    >> msgid: index-pack died"
level=warning msg="[po/bg.po]    >> msgstr: командата „git index-pack“ не завърши успешно"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: s.merge"
level=warning msg="[po/bg.po]    >> msgid: invalid branch.%s.merge; cannot rebase onto > 1 branch"
level=warning msg="[po/bg.po]    >> msgstr: неправилен клон за сливане „%s“. Невъзможно е да пребазирате върху повече от 1 клон"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: new_index"
level=warning msg="[po/bg.po]    >> msgid: merge: Unable to write new index file"
level=warning msg="[po/bg.po]    >> msgstr: сливане: новият индекс не може да бъде запазен"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --config="
level=warning msg="[po/bg.po]    >> msgid: missing --config=<config>"
level=warning msg="[po/bg.po]    >> msgstr: липсва --config=НАСТРОЙКА"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --extcmd="
level=warning msg="[po/bg.po]    >> msgid: no <cmd> given for --extcmd=<cmd>"
level=warning msg="[po/bg.po]    >> msgstr: не е зададена команда за „--extcmd=КОМАНДА“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --tool="
level=warning msg="[po/bg.po]    >> msgid: no <tool> given for --tool=<tool>"
level=warning msg="[po/bg.po]    >> msgstr: не е зададена програма за „--tool=ПРОГРАМА“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: git pack-objects"
level=warning msg="[po/bg.po]    >> msgid: pack-objects died"
level=warning msg="[po/bg.po]    >> msgstr: Командата „git pack-objects“ не завърши успешно"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: git rev-list"
level=warning msg="[po/bg.po]    >> msgid: ref '%s' is excluded by the rev-list options"
level=warning msg="[po/bg.po]    >> msgstr: указателят „%s“ не е бил включен поради опциите зададени на „git rev-list“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: run_command"
level=warning msg="[po/bg.po]    >> msgid: run_command returned non-zero status for %s\n."
level=warning msg="[po/bg.po]    >> msgstr: изпълнената команда завърши с ненулев изход за „%s“\n."
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: run_command"
level=warning msg="[po/bg.po]    >> msgid: run_command returned non-zero status while recursing in the nested submodules of %s\n."
level=warning msg="[po/bg.po]    >> msgstr: изпълнената команда завърши с ненулев изход при обхождане на подмодулите, вложени в „%s“\n."
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --raw, --stat"
level=warning msg="[po/bg.po]    >> msgid: synonym for '-p --raw'"
level=warning msg="[po/bg.po]    >> msgstr: псевдоним на „-p --stat“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: git cherry-pick"
level=warning msg="[po/bg.po]    >> msgid: try \"git revert (--continue | %s--abort | --quit)\""
level=warning msg="[po/bg.po]    >> msgstr: използвайте „git cherry-pick (--continue | %s--abort | --quit)“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: lazy_dir"
level=warning msg="[po/bg.po]    >> msgid: unable to create lazy_dir thread: %s"
level=warning msg="[po/bg.po]    >> msgstr: не може да се създаде нишка за директории: %s"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: lazy_name"
level=warning msg="[po/bg.po]    >> msgid: unable to create lazy_name thread: %s"
level=warning msg="[po/bg.po]    >> msgstr: не може да се създаде нишка за имена: %s"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: lazy_name"
level=warning msg="[po/bg.po]    >> msgid: unable to join lazy_name thread: %s"
level=warning msg="[po/bg.po]    >> msgstr: не може да се изчака нишка за имена: %s"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --patch"
level=warning msg="[po/bg.po]    >> msgid: unknown --patch mode: %s"
level=warning msg="[po/bg.po]    >> msgstr: неизвестна стратегия за прилагане на кръпка: „%s“"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --mirror"
level=warning msg="[po/bg.po]    >> msgid: unknown mirror argument: %s"
level=warning msg="[po/bg.po]    >> msgstr: неправилна стойност за „--mirror“: %s"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: update_ref"
level=warning msg="[po/bg.po]    >> msgid: update_ref failed for ref '%s': %s"
level=warning msg="[po/bg.po]    >> msgstr: неуспешно обновяване на указателя „%s“: %s"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: git merge-base"
level=warning msg="[po/bg.po]    >> msgid: use 'merge-base --fork-point' to refine upstream"
level=warning msg="[po/bg.po]    >> msgstr: за доуточняването на следения клон, използвайте:\n\n git merge-base --fork-point"
level=warning
level=warning msg="[po/bg.po]    mismatch variable names: --schedule="
level=warning msg="[po/bg.po]    >> msgid: use at most one of --auto and --schedule=<frequency>"
level=warning msg="[po/bg.po]    >> msgstr: може да се указва максимум една от опциите „--auto“ и „--schedule=ЧЕСТОТА“"
level=warning
EOF

test_expect_success "check typos in bg.po" '
	git -C workdir $HELPER check-po bg >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_failure "no typos in master branch" '
	git -C workdir checkout master &&
	git -C workdir $HELPER \
		check-po --report-typos-as-errors bg
'

test_done
