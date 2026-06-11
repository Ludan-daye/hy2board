package service

import (
	"fmt"
	"net"
	"regexp"
	"sort"
	"strings"

	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/gorm"
)

var domainRuleValuePattern = regexp.MustCompile(`^[A-Za-z0-9*._-]+$`)

func ValidateCustomRoutingRule(rule *model.CustomRoutingRule) error {
	rule.Name = strings.TrimSpace(rule.Name)
	rule.Kind = strings.ToUpper(strings.TrimSpace(rule.Kind))
	rule.Value = strings.TrimSpace(rule.Value)
	rule.Policy = normalizeRulePolicy(rule.Policy)
	rule.Note = strings.TrimSpace(rule.Note)

	if rule.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !allowedRuleKinds()[RuleKind(rule.Kind)] {
		return fmt.Errorf("invalid rule kind")
	}
	if !allowedRulePolicies()[rule.Policy] {
		return fmt.Errorf("invalid rule policy")
	}
	if rule.Value == "" {
		return fmt.Errorf("value is required")
	}

	switch RuleKind(rule.Kind) {
	case RuleDomain, RuleDomainSuffix:
		if strings.Contains(rule.Value, "://") || strings.ContainsAny(rule.Value, ", \t\r\n") || !domainRuleValuePattern.MatchString(rule.Value) {
			return fmt.Errorf("invalid domain value")
		}
		rule.Value = strings.TrimPrefix(strings.ToLower(rule.Value), ".")
	case RuleDomainKeyword:
		if strings.ContainsAny(rule.Value, ", \t\r\n") {
			return fmt.Errorf("invalid keyword value")
		}
		rule.Value = strings.ToLower(rule.Value)
	case RuleIPCIDR:
		if _, _, err := net.ParseCIDR(rule.Value); err != nil {
			return fmt.Errorf("invalid ip cidr value")
		}
	case RuleGeoIP:
		rule.Value = strings.ToUpper(rule.Value)
		if !regexp.MustCompile(`^[A-Z]{2}$`).MatchString(rule.Value) {
			return fmt.Errorf("invalid geoip value")
		}
	}
	return nil
}

func ListEnabledCustomRoutingRules(db *gorm.DB) ([]RoutingRule, error) {
	var rows []model.CustomRoutingRule
	if err := db.Where("enabled = ?", true).Order("sort_order asc, id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	rules := make([]RoutingRule, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, CustomRoutingRuleToRoutingRule(row))
	}
	return rules, nil
}

func CustomRoutingRuleToRoutingRule(row model.CustomRoutingRule) RoutingRule {
	return RoutingRule{
		Kind:   RuleKind(row.Kind),
		Value:  row.Value,
		Policy: row.Policy,
	}
}

func ResolveRoutingRuleLine(rule RoutingRule, hasAI bool, hasHongKong bool) string {
	rule.Policy = resolveRulePolicy(rule.Policy, hasAI, hasHongKong)
	return rule.Line()
}

func AllowedRoutingRuleKinds() []string {
	items := make([]string, 0, len(allowedRuleKinds()))
	for k := range allowedRuleKinds() {
		items = append(items, string(k))
	}
	sort.Strings(items)
	return items
}

func AllowedRoutingRulePolicies() []string {
	items := make([]string, 0, len(allowedRulePolicies()))
	for p := range allowedRulePolicies() {
		items = append(items, p)
	}
	sort.Strings(items)
	return items
}

func normalizeRulePolicy(policy string) string {
	compact := strings.ReplaceAll(strings.TrimSpace(policy), " ", "")
	switch strings.ToUpper(compact) {
	case "AUTO":
		return GroupAuto
	case "HONGKONG", "HK":
		return GroupHongKong
	case "AI":
		return GroupAI
	case "STREAMING":
		return GroupStreaming
	case "GLOBAL":
		return GroupGlobal
	case "DIRECT":
		return GroupDirect
	case "REJECT":
		return GroupReject
	default:
		return strings.TrimSpace(policy)
	}
}

func allowedRuleKinds() map[RuleKind]bool {
	return map[RuleKind]bool{
		RuleDomain:        true,
		RuleDomainSuffix:  true,
		RuleDomainKeyword: true,
		RuleIPCIDR:        true,
		RuleGeoIP:         true,
	}
}

func allowedRulePolicies() map[string]bool {
	return map[string]bool{
		GroupAuto:      true,
		GroupHongKong:  true,
		GroupAI:        true,
		GroupStreaming: true,
		GroupGlobal:    true,
		GroupDirect:    true,
		GroupReject:    true,
	}
}
