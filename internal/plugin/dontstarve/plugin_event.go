package dontstarve

import (
	"fmt"
	"regexp"
)

type EventType int

const (
	ETServerStart EventType = iota
	ETPlayerJoin
	ETPlayerLeave
	ETPlayerDeath
	ETPlayerResurrect
	ETPlayerServerAnnouncement
)

type eventRegexp struct {
	regexp        *regexp.Regexp
	regexpString  string
	messageFormat string
	EventType     EventType
}

func (e *eventRegexp) Search(b []byte) []byte {
	res := e.regexp.Find(b)
	if len(res) == 0 {
		return nil
	}
	return fmt.Appendf(nil, e.messageFormat, res[1:])
}

var eventRegexps = func() []*eventRegexp {
	v := []*eventRegexp{
		{
			// Shard server started
			regexpString:  `Shard server started`,
			messageFormat: "服务器启动成功",
		},
		{
			//
			regexpString:  `\[Join Announcement\] (.*)`,
			messageFormat: "%s 加入游戏",
		},
		{
			//
			regexpString:  `\[Leave Announcement\] (.*)`,
			messageFormat: "%s 离开游戏",
		},
		{
			//
			regexpString:  `\[Death Announcement\] (.*) 死于： (.*)。`,
			messageFormat: "%s 死于: %s",
		},
		{
			//
			regexpString:  `\[Resurrect Announcement\] (.*) 复活自： (.*).`,
			messageFormat: "%s 复活自: %s",
		},
		{
			//
			regexpString:  `\[Announcement\] (.*)`,
			messageFormat: "全服宣告: %s",
		},
	}
	for i := range v {
		v[i].regexp = regexp.MustCompile(v[i].regexpString)
		v[i].EventType = EventType(i)
	}
	return v
}()
