package query

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// L2Result L2语义标签推断结果
type L2Result struct {
	Event      []string `json:"event,omitempty"`
	Activity   []string `json:"activity,omitempty"`
	Atmosphere []string `json:"atmosphere,omitempty"`
	Attributes []string `json:"attributes,omitempty"`
	SceneType  []string `json:"scene_type,omitempty"`
}

// L2Engine L2推断引擎
type L2Engine struct {
	ExactRules        map[string]L2Result `json:"exact_rules"`
	CombinationRules  []struct {
		If   []string `json:"if"`
		Then L2Result `json:"then"`
	} `json:"combination_rules"`
	cache map[string]L2Result
}

// NewL2Engine 创建新的L2推断引擎
func NewL2Engine(path string) (*L2Engine, error) {
	if path == "" {
		path = "config/l2_rules_v1.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取L2规则失败: %w", err)
	}
	var engine L2Engine
	if err := json.Unmarshal(data, &engine); err != nil {
		return nil, fmt.Errorf("解析L2规则失败: %w", err)
	}
	engine.cache = make(map[string]L2Result)
	return &engine, nil
}

// Infer 基于L1标签推断L2语义标签
func (e *L2Engine) Infer(l1Tags []string) L2Result {
	if len(l1Tags) == 0 {
		return L2Result{}
	}
	// 排序后生成缓存key
	sorted := make([]string, len(l1Tags))
	copy(sorted, l1Tags)
	sort.Strings(sorted)
	cacheKey := fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(sorted, ","))))
	if cached, ok := e.cache[cacheKey]; ok {
		return cached
	}

	result := L2Result{}
	setMap := make(map[string]map[string]bool) // field -> value -> exists

	add := func(field string, values []string) {
		if len(values) == 0 {
			return
		}
		if setMap[field] == nil {
			setMap[field] = make(map[string]bool)
		}
		for _, v := range values {
			setMap[field][v] = true
		}
	}

	// 1. Exact rules
	for _, tag := range l1Tags {
		if rule, ok := e.ExactRules[tag]; ok {
			add("event", rule.Event)
			add("activity", rule.Activity)
			add("atmosphere", rule.Atmosphere)
			add("attributes", rule.Attributes)
			add("scene_type", rule.SceneType)
		}
	}

	// 2. Combination rules
	l1Set := make(map[string]bool)
	for _, t := range l1Tags {
		l1Set[t] = true
	}
	for _, rule := range e.CombinationRules {
		matched := true
		for _, req := range rule.If {
			if !l1Set[req] {
				matched = false
				break
			}
		}
		if matched {
			add("event", rule.Then.Event)
			add("activity", rule.Then.Activity)
			add("atmosphere", rule.Then.Atmosphere)
			add("attributes", rule.Then.Attributes)
			add("scene_type", rule.Then.SceneType)
		}
	}

	// 3. 构建结果
	for k, v := range setMap["event"] {
		if v {
			result.Event = append(result.Event, k)
		}
	}
	for k, v := range setMap["activity"] {
		if v {
			result.Activity = append(result.Activity, k)
		}
	}
	for k, v := range setMap["atmosphere"] {
		if v {
			result.Atmosphere = append(result.Atmosphere, k)
		}
	}
	for k, v := range setMap["attributes"] {
		if v {
			result.Attributes = append(result.Attributes, k)
		}
	}
	for k, v := range setMap["scene_type"] {
		if v {
			result.SceneType = append(result.SceneType, k)
		}
	}

	// 排序保证稳定性
	sort.Strings(result.Event)
	sort.Strings(result.Activity)
	sort.Strings(result.Atmosphere)
	sort.Strings(result.Attributes)
	sort.Strings(result.SceneType)

	e.cache[cacheKey] = result
	return result
}
