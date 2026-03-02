#!/bin/sh

test_description="test git-po-helper team"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	(
		cd workdir &&
		git switch po-2.31.1 &&
		test -f po/TEAMS
	)
'

test_expect_success "check syntax of po/TEAMS" '
	test_must_fail git -C workdir $HELPER team --check >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	level=error msg="bad syntax at po/TEAMS:79 (unknown key \"Respository\"): Respository:    https://github.com/l10n-tw/git-po"
	level=error msg="bad syntax at po/TEAMS:80 (need two tabs between k/v): Leader:     Yi-Jyun Pan <pan93412 AT gmail.com>"
	ERROR: team command failed
	EOF

	test_cmp expect actual
'

test_expect_success "fixed po/TEAMS" '
	(
		cd workdir &&

		sed -e "s/^Respository:/Repository:/" \
			-e "s/^Leader: 	/Leader:		/" <po/TEAMS >po/TEAMS.new &&
		mv po/TEAMS.new po/TEAMS
	) &&

	git -C workdir $HELPER team --check >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	EOF

	test_cmp expect actual
'

test_expect_success "show teams" '
	git -C workdir $HELPER team >out &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	bg (Bulgarian)
	ca (Catalan)
	de (German)
	el (Greek)
	es (Spanish)
	fr (French)
	id (Indonesian)
	it (Italian)
	ko (Korean)
	pl (Polish)
	pt_PT (Portuguese - Portugal)
	ru (Russian)
	sv (Swedish)
	tr (Turkish)
	vi (Vietnamese)
	zh_CN (Simplified Chinese)
	zh_TW (Traditional Chinese)
	EOF

	test_cmp expect actual
'

test_expect_success "show team leaders" '
	git -C workdir $HELPER team --leader >out &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-\EOF &&
	Alexander Shopov <ash@kambanaria.org>
	Jordi Mas <jmas@softcatala.org>
	Matthias Rüster <matthias.ruester@gmail.com>
	Jimmy Angelakos <vyruss@hellug.gr>
	Christopher Díaz <christopher.diaz.riv@gmail.com>
	Jean-Noël Avila <jn.avila@free.fr>
	Bagas Sanjaya <bagasdotme@gmail.com>
	Alessandro Menti <alessandro.menti@alessandromenti.it>
	Gwan-gyeong Mun <elongbug@gmail.com>
	Arusekk <arek_koz@o2.pl>
	Daniel Santos <hello@brighterdan.com>
	Dimitriy Ryazantcev <DJm00n@mail.ru>
	Peter Krefting <peter@softwolves.pp.se>
	Emir SARI <bitigchi@me.com>
	Trần Ngọc Quân <vnwildman@gmail.com>
	Jiang Xin <worldhello.net@gmail.com>
	Yi-Jyun Pan <pan93412@gmail.com>
	EOF

	test_cmp expect actual
'

test_expect_success "show team members" '
	git -C workdir $HELPER team --members >out &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	Alex Henrie <alexhenrie24@gmail.com>
	Ralf Thielow <ralf.thielow@gmail.com>
	Phillip Szelat <phillip.szelat@gmail.com>
	Sébastien Helleu <flashcode@flashtux.org>
	Changwoo Ryu <cwryu@debian.org>
	Sihyeon Jang <uneedsihyeon@gmail.com>
	insolor <insolor@gmail.com>
	Nguyễn Thái Ngọc Duy <pclouds@gmail.com>
	Ray Chen <oldsharp@gmail.com>
	依云 <lilydjwg@gmail.com>
	Fangyi Zhou <me@fangyi.io>
	Franklin Weng <franklin@goodhorse.idv.tw>
	EOF

	test_cmp expect actual
'

test_expect_success "show all team members (leader + members)" '
	git -C workdir $HELPER team --all >out &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	Alexander Shopov <ash@kambanaria.org>
	Jordi Mas <jmas@softcatala.org>
	Alex Henrie <alexhenrie24@gmail.com>
	Matthias Rüster <matthias.ruester@gmail.com>
	Ralf Thielow <ralf.thielow@gmail.com>
	Phillip Szelat <phillip.szelat@gmail.com>
	Jimmy Angelakos <vyruss@hellug.gr>
	Christopher Díaz <christopher.diaz.riv@gmail.com>
	Jean-Noël Avila <jn.avila@free.fr>
	Sébastien Helleu <flashcode@flashtux.org>
	Bagas Sanjaya <bagasdotme@gmail.com>
	Alessandro Menti <alessandro.menti@alessandromenti.it>
	Gwan-gyeong Mun <elongbug@gmail.com>
	Changwoo Ryu <cwryu@debian.org>
	Sihyeon Jang <uneedsihyeon@gmail.com>
	Arusekk <arek_koz@o2.pl>
	Daniel Santos <hello@brighterdan.com>
	Dimitriy Ryazantcev <DJm00n@mail.ru>
	insolor <insolor@gmail.com>
	Peter Krefting <peter@softwolves.pp.se>
	Emir SARI <bitigchi@me.com>
	Trần Ngọc Quân <vnwildman@gmail.com>
	Nguyễn Thái Ngọc Duy <pclouds@gmail.com>
	Jiang Xin <worldhello.net@gmail.com>
	Ray Chen <oldsharp@gmail.com>
	依云 <lilydjwg@gmail.com>
	Fangyi Zhou <me@fangyi.io>
	Yi-Jyun Pan <pan93412@gmail.com>
	Franklin Weng <franklin@goodhorse.idv.tw>
	EOF

	test_cmp expect actual
'

test_done
