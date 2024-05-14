package fsnet

import (
	"net/url"
	"strings"

	bypass2 "github.com/go-gost/core/bypass"
	"github.com/go-gost/x/bypass"
	"github.com/go-gost/x/logger"
)

// https://gost.run/concepts/bypass/
//
// 注意: 通配符 `*` 只适用于域名
func ParseBypass(query string) bypass2.Bypass {
	q, _ := url.ParseQuery(query)
	rules := q["bypass"]
	bps := make([]bypass2.Bypass, len(rules))
	for i, rule := range rules {
		whitelist := strings.HasPrefix(rule, "~")
		if whitelist {
			rule = strings.TrimPrefix(rule, "~")
		}
		l := []string{}
		for _, v := range strings.Split(rule, ",") {
			v := strings.TrimSpace(v)
			if v == "" {
				continue
			}
			l = append(l, v)
		}
		bps[i] = bypass.NewBypass(
			bypass.MatchersOption(l),
			bypass.LoggerOption(logger.Nop()),
			bypass.WhitelistOption(whitelist),
		)
	}
	return bypass2.BypassGroup(bps...)
}
