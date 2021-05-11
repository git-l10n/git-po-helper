#!/bin/sh

test_description="test git-po-helper team"

. ./lib/sharness.sh

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	test -f workdir/po/TEAMS
'

test_expect_success "check syntax of po/TEAMS" '
	(
		cd workdir &&

		cat >expect <<-EOF &&
		level=error msg="bad syntax at line 79 (unknown key \"Respository\"): Respository:\thttps://github.com/l10n-tw/git-po"
		level=error msg="bad syntax at line 80 (need two tabs between k/v): Leader: \tYi-Jyun Pan <pan93412 AT gmail.com>"
		EOF
		test_must_fail git-po-helper team --check >actual 2>&1 &&
		test_cmp expect actual
	)
'

test_expect_success "fixed po/TEAMS" '
	(
		cd workdir &&

		sed -e "s/^Respository:/Repository:/" \
			-e "s/^Leader: 	/Leader:		/" <po/TEAMS >po/TEAMS.new &&
		mv po/TEAMS.new po/TEAMS &&

		cat >expect <<-EOF &&
		EOF
		git-po-helper team --check >actual 2>&1 &&
		test_cmp expect actual
	)
'

test_expect_success "show teams" '
	(
		cd workdir &&

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
		git-po-helper team >actual 2>error &&
		test_cmp expect actual &&
		test ! -s error
	)
'

test_expect_success "show team leaders" '
	(
		cd workdir &&

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
		git-po-helper team --leader >actual 2>error &&
		test_cmp expect actual &&
		test ! -s error
	)
'

test_expect_success "show team members" '
	(
		cd workdir &&

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

		git-po-helper team --members >actual 2>error &&
		test_cmp expect actual &&
		test ! -s error
	)
'

test_done
