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

// User contains user name and email.
type User struct {
	Name  string
	Email string
}

// Team contains infomation for a l10n team.
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

// ParseTeams implements parse of "po/TEAMS" file.
func ParseTeams(fileName string) ([]Team, []error) {
	var (
		teams  []Team
		team   Team
		nr     = 0
		errors = []error{}
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
			errors = append(errors,
				fmt.Errorf(`invalid utf-8 in: %s`, line))
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			if isHead {
				continue
			} else {
				errors = append(errors,
					fmt.Errorf(`bad syntax at po/TEAMS:%d (no column): %s`,
						nr, line))
				break
			}
		}
		if len(kv[1]) < 2 {
			errors = append(errors,
				fmt.Errorf(`bad syntax at po/TEAMS:%d (too short value): %s`,
					nr, line))
		} else if kv[0] == "Leader" { // Skip two tabs
			if kv[1][0] != '\t' || kv[1][1] != '\t' {
				errors = append(errors,
					fmt.Errorf(`bad syntax at po/TEAMS:%d (need two tabs between k/v): %s`,
						nr, line))
				kv[1] = strings.TrimSpace(kv[1])
			} else {
				kv[1] = kv[1][2:]
			}
		} else { // skip one tab
			if kv[1][0] != '\t' {
				errors = append(errors,
					fmt.Errorf(`bad syntax at po/TEAMS:%d (need tab between k/v): %s`,
						nr, line))
				kv[1] = strings.TrimSpace(kv[1])
			} else {
				kv[1] = kv[1][1:]
			}
		}
		if strings.TrimSpace(kv[1]) != kv[1] {
			errors = append(errors,
				fmt.Errorf(`bad syntax at po/TEAMS:%d (too many spaces): %s`,
					nr, line))
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
				errors = append(errors,
					fmt.Errorf(`bad syntax at po/TEAMS:%d (fail to parse user): %s`,
						nr, line),
					fmt.Errorf("\t%s", err))
			} else {
				team.Leader = user
			}
		case "Members":
			user, err := parseUser(kv[1])
			if err != nil {
				errors = append(errors,
					fmt.Errorf(`bad syntax at po/TEAMS:%d (fail to parse user): %s`,
						nr, line),
					fmt.Errorf("\t%s", err))
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
					errors = append(errors,
						fmt.Errorf(`bad syntax at po/TEAMS:%d (fail to parse user): %s`,
							nr, line))
				} else {
					team.Members = append(team.Members, user)
				}
			}
		default:
			if isHead {
				continue
			} else {
				errors = append(errors,
					fmt.Errorf(`bad syntax at po/TEAMS:%d (unknown key "%s"): %s`,
						nr, kv[0], line))
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
	return teams, errors
}

// ShowTeams will show leader/members of a team.
func ShowTeams(args ...string) bool {
	var (
		teams       []Team
		errors      []error
		optLeader   = viper.GetBool("team-leader")
		optMembers  = viper.GetBool("team-members")
		optAll      = viper.GetBool("all-team-members")
		optLanguage = viper.GetBool("show-language")
		ret         = true
	)
	teams, errors = ParseTeams("")
	if len(errors) != 0 {
		for _, error := range errors {
			log.Error(error)
		}
		ret = false
	}
	log.Debugf(`get %d teams from "po/TEAMS"`, len(teams))
	if viper.GetBool("team-check") {
		return ret
	}
	for _, team := range teams {
		prefix := ""
		if optLanguage {
			fmt.Printf("# %s:\n", team.Language)
			prefix = "\t"
		}
		if (optLeader || optAll) && team.Leader.Name != "" {
			fmt.Printf("%s%s <%s>\n", prefix, team.Leader.Name, team.Leader.Email)
		}
		// If no maintainer, not show members
		if (optMembers && team.Leader.Name != "") || optAll {
			for _, member := range team.Members {
				fmt.Printf("%s%s <%s>\n", prefix, member.Name, member.Email)
			}
		}
		if !optLeader && !optMembers && !optAll && !optLanguage {
			fmt.Printf("# %s:\n", team.Language)
		}
	}
	return ret
}
