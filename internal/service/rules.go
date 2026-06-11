package service

import (
	"fmt"

	"github.com/ludandaye/hy2board/internal/model"
)

// Rule group names used by generators as proxy-group names.
const (
	GroupAuto      = "Auto"
	GroupManual    = "Manual"
	GroupAI        = "AI"
	GroupStreaming = "Streaming"
	GroupHongKong  = "HongKong"
	GroupGlobal    = "Global"
	GroupDirect    = "DIRECT"
	GroupReject    = "REJECT"
)

type RuleKind string

const (
	RuleDomain        RuleKind = "DOMAIN"
	RuleDomainSuffix  RuleKind = "DOMAIN-SUFFIX"
	RuleDomainKeyword RuleKind = "DOMAIN-KEYWORD"
	RuleIPCIDR        RuleKind = "IP-CIDR"
	RuleGeoIP         RuleKind = "GEOIP"
)

// RoutingRule is one client rule before it is rendered into Clash/Surge syntax.
type RoutingRule struct {
	Kind   RuleKind
	Value  string
	Policy string
}

func (r RoutingRule) Line() string {
	return fmt.Sprintf("%s,%s,%s", r.Kind, r.Value, r.Policy)
}

func (r RoutingRule) withPolicy(policy string) RoutingRule {
	r.Policy = policy
	return r
}

// RuleSpec is a bundle of curated rules for one route category.
type RuleSpec struct {
	GroupName string
	Rules     []RoutingRule
}

var (
	adBlockRules = RuleSpec{
		GroupName: GroupReject,
		Rules: rules(GroupReject,
			"doubleclick.net",
			"googleadservices.com",
			"googlesyndication.com",
			"google-analytics.com",
			"adnxs.com",
			"scorecardresearch.com",
		),
	}

	aiRules = RuleSpec{
		GroupName: GroupAI,
		Rules: append(
			rules(GroupAI,
				"openai.com",
				"chatgpt.com",
				"oaistatic.com",
				"oaiusercontent.com",
				"anthropic.com",
				"claude.ai",
				"perplexity.ai",
				"poe.com",
				"x.ai",
				"grok.com",
				"cursor.com",
				"cursor.sh",
				"ping0.cc",
			),
			exact(GroupAI,
				"gemini.google.com",
				"ai.google.dev",
				"generativelanguage.googleapis.com",
			)...,
		),
	}

	streamingRules = RuleSpec{
		GroupName: GroupStreaming,
		Rules: rules(GroupStreaming,
			"netflix.com",
			"nflxvideo.net",
			"nflxso.net",
			"nflxext.com",
			"nflximg.net",
			"disneyplus.com",
			"disney.com",
			"dssott.com",
			"youtube.com",
			"youtu.be",
			"googlevideo.com",
			"ytimg.com",
			"spotify.com",
			"scdn.co",
			"hulu.com",
			"hbomax.com",
			"max.com",
		),
	}

	googleScholarRules = RuleSpec{
		GroupName: GroupHongKong,
		Rules: rules(GroupHongKong,
			"scholar.google.com",
		),
	}

	globalRules = RuleSpec{
		GroupName: GroupGlobal,
		Rules: append(
			rules(GroupGlobal,
				"github.com",
				"githubusercontent.com",
				"githubassets.com",
				"github.io",
				"githubcopilot.com",
				"telegram.org",
				"telegram.me",
				"t.me",
				"tdesktop.com",
				"docker.com",
				"docker.io",
				"npmjs.com",
				"npmjs.org",
				"golang.org",
				"go.dev",
				"microsoft.com",
				"microsoftonline.com",
				"live.com",
				"windows.net",
				"speedtest.net",
				"ooklaserver.net",
				"fast.com",
			),
			exact(GroupGlobal,
				"proxy.golang.org",
				"registry.npmjs.org",
				"copilot.microsoft.com",
			)...,
		),
	}

	chinaDirectRules = RuleSpec{
		GroupName: GroupDirect,
		Rules: append(
			rules(GroupDirect,
				"cn",
				"baidu.com",
				"bdstatic.com",
				"qq.com",
				"qlogo.cn",
				"gtimg.cn",
				"wechat.com",
				"taobao.com",
				"tmall.com",
				"tb.cn",
				"jd.com",
				"360buyimg.com",
				"alipay.com",
				"aliyun.com",
				"aliyuncs.com",
				"bilibili.com",
				"bilivideo.com",
				"zhihu.com",
				"weibo.com",
				"douyin.com",
				"byteimg.com",
				"bytedance.com",
				"toutiao.com",
				"amap.com",
				"gaode.com",
				"qcloud.com",
				"myqcloud.com",
				"tencentcloud.com",
				"huaweicloud.com",
				"volcengine.com",
				"baidubce.com",
				"pinduoduo.com",
				"xiaohongshu.com",
				"meituan.com",
				"dianping.com",
				"163.com",
				"netease.com",
				"mi.com",
				"xiaomi.com",
			),
			keywords(GroupDirect, "baidu", "taobao", "bilibili")...,
		),
	}

	chinaGeoIPRule = RoutingRule{Kind: RuleGeoIP, Value: "CN", Policy: GroupDirect}
)

// RuleSet is the computed set of rules + group flags for one user's subscription.
type RuleSet struct {
	HasAI        bool
	HasStreaming bool
	HasGlobal    bool
	HasChina     bool
	HasAdBlock   bool
	Rules        []RoutingRule
}

// BuildRuleSet composes a RuleSet from a user's rule_* flags, with legacy chain_proxy fallback.
func BuildRuleSet(u model.User, customRules ...RoutingRule) RuleSet {
	rs := RuleSet{}
	if u.RuleAdBlock {
		rs.HasAdBlock = true
		rs.Rules = append(rs.Rules, adBlockRules.Rules...)
	}
	for _, rule := range customRules {
		switch rule.Policy {
		case GroupAI:
			rs.HasAI = true
		case GroupStreaming:
			rs.HasStreaming = true
		case GroupGlobal:
			rs.HasGlobal = true
		}
	}
	rs.Rules = append(rs.Rules, customRules...)
	if u.RuleAI || u.ChainProxy {
		rs.HasAI = true
		rs.Rules = append(rs.Rules, aiRules.Rules...)
	}
	if u.RuleStreaming {
		rs.HasStreaming = true
		rs.Rules = append(rs.Rules, streamingRules.Rules...)
	}

	rs.Rules = append(rs.Rules, googleScholarRules.Rules...)

	rs.HasGlobal = true
	rs.Rules = append(rs.Rules, globalRules.Rules...)

	if u.RuleChina {
		rs.HasChina = true
		rs.Rules = append(rs.Rules, chinaDirectRules.Rules...)
		rs.Rules = append(rs.Rules, chinaGeoIPRule)
	}
	return rs
}

func (rs RuleSet) Lines(aiPolicy string, hongKongPolicy string) []string {
	lines := make([]string, 0, len(rs.Rules))
	for _, rule := range rs.Rules {
		rule = rule.withPolicy(resolveRulePolicyWithNames(rule.Policy, aiPolicy, hongKongPolicy))
		lines = append(lines, rule.Line())
	}
	return lines
}

func resolveRulePolicy(policy string, hasAI bool, hasHongKong bool) string {
	aiPolicy := GroupAuto
	if hasAI {
		aiPolicy = GroupAI
	}
	hongKongPolicy := GroupAuto
	if hasHongKong {
		hongKongPolicy = GroupHongKong
	}
	return resolveRulePolicyWithNames(policy, aiPolicy, hongKongPolicy)
}

func resolveRulePolicyWithNames(policy string, aiPolicy string, hongKongPolicy string) string {
	switch policy {
	case GroupAI:
		return aiPolicy
	case GroupHongKong:
		return hongKongPolicy
	default:
		return policy
	}
}

func rules(policy string, domains ...string) []RoutingRule {
	items := make([]RoutingRule, 0, len(domains))
	for _, domain := range domains {
		items = append(items, RoutingRule{
			Kind:   RuleDomainSuffix,
			Value:  domain,
			Policy: policy,
		})
	}
	return items
}

func exact(policy string, domains ...string) []RoutingRule {
	items := make([]RoutingRule, 0, len(domains))
	for _, domain := range domains {
		items = append(items, RoutingRule{
			Kind:   RuleDomain,
			Value:  domain,
			Policy: policy,
		})
	}
	return items
}

func keywords(policy string, words ...string) []RoutingRule {
	items := make([]RoutingRule, 0, len(words))
	for _, word := range words {
		items = append(items, RoutingRule{
			Kind:   RuleDomainKeyword,
			Value:  word,
			Policy: policy,
		})
	}
	return items
}
