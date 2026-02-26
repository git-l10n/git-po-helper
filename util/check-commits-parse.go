package util

import (
	"bufio"
	"io"
	"strings"

	log "github.com/sirupsen/logrus"
)

type commitLog struct {
	// Meta holds header of a raw commit
	Meta map[string]interface{}
	// Msg holds commit message of a raw commit
	Msg []string
	// oid is commit ID for this commit
	oid string
}

func newCommitLog(oid string) commitLog {
	commitLog := commitLog{oid: oid}
	commitLog.Meta = make(map[string]interface{})
	commitLog.Msg = []string{}
	return commitLog
}

// Encoding is encoding for this commit log
func (v *commitLog) Encoding() string {
	if e, ok := v.Meta["encoding"]; ok {
		return e.(string)
	}
	return defaultEncoding
}

// CommitID is commit-id for this commit log
func (v *commitLog) CommitID() string {
	if len(v.oid) > 7 {
		return v.oid[:7]
	}
	return v.oid
}

func (v *commitLog) isMergeCommit() bool {
	if parents, ok := v.Meta["parent"]; ok {
		return len(parents.([]string)) > 1
	}
	return false
}

func (v *commitLog) hasGpgSig() bool {
	if val, ok := v.Meta["gpgsig"]; ok {
		return val.(bool)
	} else if val, ok := v.Meta["gpgsig-sha256"]; ok {
		return val.(bool)
	}
	return false
}

// Parse reads and parse raw commit object
func (v *commitLog) Parse(r io.Reader) bool {
	var (
		ret    = true
		isMeta = true
	)

	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		if line == "" {
			break
		}
		line = strings.TrimRight(line, "\n")
		if isMeta && line == "" {
			isMeta = false
			continue
		}
		if isMeta {
			kv := strings.SplitN(line, " ", 2)
			if len(kv) != 2 {
				log.Errorf("commit %s: cannot parse commit HEADER: %s", v.CommitID(), line)
				ret = false
			}
			switch kv[0] {
			case "author", "committer", "encoding", "tree":
				if _, ok := v.Meta[kv[0]]; ok {
					log.Errorf("commit %s: duplicate header: %s", v.CommitID(), line)
					ret = false
				} else {
					v.Meta[kv[0]] = kv[1]
				}
			case "parent":
				if _, ok := v.Meta[kv[0]]; !ok {
					v.Meta[kv[0]] = []string{}
				}
				v.Meta[kv[0]] = append(v.Meta[kv[0]].([]string), kv[1])
			case "gpgsig", "gpgsig-sha256", "mergetag":
				if _, ok := v.Meta[kv[0]]; ok {
					log.Errorf("commit %s: duplicate header: %s", v.CommitID(), line)
					ret = false
					break
				}
				v.Meta[kv[0]] = true
				for {
					peek, err := reader.Peek(1)
					if err == nil {
						if peek[0] == ' ' {
							// Consume one line
							_, err = reader.ReadString('\n')
						} else {
							// Next header
							break
						}
					}
					if err != nil {
						log.Errorf(`commit %s: header "%s" is too short, early EOF: %s`,
							v.CommitID(), kv[0], err)
						ret = false
						break
					}
				}
			default:
				log.Errorf("commit %s: unknown commit header: %s", v.CommitID(), line)
				ret = false
			}
		} else {
			v.Msg = append(v.Msg, line)
		}
		if err != nil {
			break
		}
	}
	return ret
}
