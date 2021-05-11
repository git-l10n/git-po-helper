package util

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	reUserEmail = regexp.MustCompile(`^(.*) <(.+@.+\..+)>`)
)

const (
	l10nTestLanguage = "is (Icelandic)"
)

type User struct {
	Name  string
	Email string
}

type Team struct {
	Language   string
	Repository string
	Leader     User
	Members    []User
}

func parseUser(line string) (User, error) {
	line = strings.Replace(line, " AT ", "@", 1)
	m := reUserEmail.FindStringSubmatch(line)
	if m == nil {
		return User{}, fmt.Errorf(`"%s" is not a valid user/email`, line)
	}
	return User{Name: m[1], Email: m[2]}, nil
}

func ParseTeams(fileName string) ([]Team, bool) {
	var (
		teams []Team
		ret   = true
		team  Team
		nr    = 0
	)

	if fileName == "" {
		fileName = filepath.Join("po", "TEAMS")
	}
	f, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("fail to open %s: %s", fileName, err)
	}
	reader := bufio.NewReader(f)
	isHead := true
	for {
		line, err := reader.ReadString('\n')
		nr++
		if line == "" && err != nil {
			break
		}
		line = strings.TrimRight(line, "\n")
		if line == "" {
			continue
		}
		if !utf8.ValidString(line) {
			log.Errorf(`invalid utf-8 in: %s`, line)
			ret = false
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			if isHead {
				continue
			} else {
				log.Errorf(`bad syntax at line %d (no column): %s`, nr, line)
				ret = false
				break
			}
		}
		if len(kv[1]) < 2 {
			log.Errorf(`bad syntax at line %d (too short value): %s`, nr, line)
			ret = false
		} else if kv[0] == "Leader" { // Skip two tabs
			if kv[1][0] != '\t' || kv[1][1] != '\t' {
				log.Errorf(`bad syntax at line %d (need two tabs between k/v): %s`, nr, line)
				ret = false
				kv[1] = strings.TrimSpace(kv[1])
			} else {
				kv[1] = kv[1][2:]
			}
		} else { // skip one tab
			if kv[1][0] != '\t' {
				log.Errorf(`bad syntax at line %d (need tab between k/v): %s`, nr, line)
				ret = false
				kv[1] = strings.TrimSpace(kv[1])
			} else {
				kv[1] = kv[1][1:]
			}
		}
		if strings.TrimSpace(kv[1]) != kv[1] {
			log.Errorf(`bad syntax at line %d (too many spaces): %s`, nr, line)
			ret = false
		}

		switch kv[0] {
		case "Language":
			// append new team, then reset
			if team.Language != "" && team.Language != l10nTestLanguage {
				teams = append(teams, team)
				team = Team{}
			}
			team.Language = kv[1]
		case "Repository":
			team.Repository = kv[1]
		case "Leader":
			user, err := parseUser(kv[1])
			if err != nil {
				log.Errorf(`bad syntax at line %d (fail to parse user): %s`, nr, line)
				log.Errorf("\t%s", err)
				ret = false
			} else {
				team.Leader = user
			}
		case "Members":
			user, err := parseUser(kv[1])
			if err != nil {
				log.Errorf(`bad syntax at line %d (fail to parse user): %s`, nr, line)
				log.Errorf("\t%s", err)
				ret = false
			} else {
				team.Members = append(team.Members, user)
			}
			for {
				buf, err := reader.Peek(2)
				if err != nil || buf[0] != '\t' || buf[1] != '\t' {
					break
				}
				line, _ := reader.ReadString('\n')
				line = line[2:]
				user, err := parseUser(line)
				if err != nil {
					log.Errorf(`bad syntax at line %d (fail to parse user): %s`, nr, line)
					ret = false
				} else {
					team.Members = append(team.Members, user)
				}
			}
		default:
			if isHead {
				continue
			} else {
				log.Errorf(`bad syntax at line %d (unknown key "%s"): %s`,
					nr, kv[0], line)
				ret = false
			}
		}
		isHead = false

		if err != nil {
			break
		}
	}
	if team.Language != "" {
		teams = append(teams, team)
	}
	return teams, ret
}

func ShowTeams(args ...string) bool {
	var (
		teams      []Team
		optLeader  = viper.GetBool("team-leader")
		optMembers = viper.GetBool("team-members")
		ret        = true
	)
	teams, ret = ParseTeams("")
	log.Debugf(`get %d teams from "po/TEAMS"`, len(teams))
	if viper.GetBool("team-check") {
		return ret
	}
	for _, team := range teams {
		if optLeader || optMembers {
			fmt.Printf("%s <%s>\n", team.Leader.Name, team.Leader.Email)
		}
		if optMembers {
			for _, member := range team.Members {
				fmt.Printf("%s <%s>\n", member.Name, member.Email)
			}
		}
		if !optLeader && !optMembers {
			fmt.Println(team.Language)
		}
	}
	return ret
}
