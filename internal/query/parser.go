package query

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParsedQuery 解析后的查询结构
type ParsedQuery struct {
	Query          string   `json:"query"`
	TimeMarkers    []string `json:"time_markers"`
	Tags           string   `json:"tags"`
	RequireAllTags bool     `json:"require_all"`
}

// ParseNaturalLanguage 使用 LLM 解析自然语言查询（OpenAI 兼容协议）
func ParseNaturalLanguage(client LLMClient, userQuery string) (*ParsedQuery, error) {
	if client == nil {
		return nil, fmt.Errorf("LLM client 未初始化")
	}

	prompt := buildPrompt(userQuery)
	respText, err := client.Generate(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}

	pq, err := extractJSON(respText)
	if err != nil {
		return nil, err
	}

	// 根据 time_markers 计算日期范围
	pq.TimeMarkers = normalizeTimeMarkers(pq.TimeMarkers)
	return pq, nil
}

func buildPrompt(query string) string {
	return fmt.Sprintf("你是照片搜索查询解析器。解析用户查询为 JSON。\n\n查询：\"%s\"\n\n输出字段：\n- query: 核心搜索词（用于语义搜索），复合查询保留原始意图\n- time_markers: 时间词数组，无则 []\n- tags: 用逗号分隔的过滤条件（用最简短的概念词，不是具体标签名）\n- require_all: 布尔值，多个条件是否需要同时满足\n\n【关键规则 - tags 必须用最简短的概念词】\n1. 花→\"花\" 人像→\"人像\" 狗→\"狗\" 猫→\"猫\" 雪→\"雪\" 海→\"海\" 日落→\"日落\" 夜景→\"夜景\" 城市→\"城市\" 建筑→\"建筑\" 飞机→\"飞机\" 车→\"车\" 美食→\"美食\" 动物→\"动物\" 山→\"山\"\n2. 复合查询（带X的Y、有X的Y、X中的Y、X里的Y）→ require_all=true，每个概念一个简短词\n   例：\"带花的人像\" → tags:\"花,人像\" require_all:true\n   例：\"雪地里的狗狗\" → tags:\"雪,狗\" require_all:true\n   例：\"有飞机的天空\" → tags:\"飞机,天空\" require_all:true\n3. 并列查询（海边日落、城市夜景等非\"带/有/中的\"）→ require_all=false\n   例：\"海边日落\" → tags:\"海,日落\" require_all:false\n   例：\"城市夜景\" → tags:\"城市,夜景\" require_all:false\n4. 单一概念 → tags单个简短词或空，require_all=false\n   例：\"花\" → tags:\"花\" require_all:false\n   例：\"故宫\" → tags:\"\" require_all:false\n\n时间词：去年/今年/上个月/上周/夏天/冬天/春天/秋天/元旦/春节\n\n示例输出：\n\"带花的人像\" → {\"query\":\"带花的人像\",\"time_markers\":[],\"tags\":\"花,人像\",\"require_all\":true}\n\"去年夏天的故宫\" → {\"query\":\"故宫\",\"time_markers\":[\"去年\",\"夏天\"],\"tags\":\"\",\"require_all\":false}\n\"海边日落\" → {\"query\":\"海边日落\",\"time_markers\":[],\"tags\":\"海,日落\",\"require_all\":false}\n\"雪地里的狗狗\" → {\"query\":\"雪地里的狗狗\",\"time_markers\":[],\"tags\":\"雪,狗\",\"require_all\":true}\n\n只输出 JSON，不要任何解释。", query)
}

func extractJSON(text string) (*ParsedQuery, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("响应中未找到 JSON: %s", text)
	}
	jsonStr := text[start : end+1]

	var pq ParsedQuery
	if err := json.Unmarshal([]byte(jsonStr), &pq); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w, raw: %s", err, jsonStr)
	}

	pq.Query = strings.TrimSpace(pq.Query)
	pq.Tags = strings.TrimSpace(pq.Tags)
	return &pq, nil
}

func normalizeTimeMarkers(markers []string) []string {
	var result []string
	for _, m := range markers {
		m = strings.TrimSpace(m)
		m = strings.ReplaceAll(m, "（", "")
		m = strings.ReplaceAll(m, "）", "")
		m = strings.ReplaceAll(m, "[", "")
		m = strings.ReplaceAll(m, "]", "")
		m = strings.ReplaceAll(m, "\"", "")
		if m != "" {
			result = append(result, m)
		}
	}
	return result
}

// ResolveDateRange 根据时间标记和当前日期计算实际的日期范围
func ResolveDateRange(markers []string) (from, to string, hasRange bool) {
	if len(markers) == 0 {
		return "", "", false
	}

	now := time.Now()
	year := now.Year()

	var startYear, endYear int
	var startMonth, endMonth time.Month = 1, 12
	var startDay, endDay int = 1, 31

	hasYear := false
	hasSeason := false
	hasMonth := false
	hasDay := false

	for _, m := range markers {
		// 年份
		if m == "去年" {
			startYear, endYear = year-1, year-1
			hasYear = true
		} else if m == "今年" {
			startYear, endYear = year, year
			hasYear = true
		} else if strings.HasSuffix(m, "年") {
			if y, err := strconv.Atoi(strings.TrimSuffix(m, "年")); err == nil && y > 1990 && y < 2100 {
				startYear, endYear = y, y
				hasYear = true
			}
		}

		// 季节
		switch m {
		case "春天", "春季":
			startMonth, endMonth = 3, 5
			startDay, endDay = 1, 31
			hasSeason = true
		case "夏天", "夏季":
			startMonth, endMonth = 6, 8
			startDay, endDay = 1, 31
			hasSeason = true
		case "秋天", "秋季":
			startMonth, endMonth = 9, 11
			startDay, endDay = 1, 30
			hasSeason = true
		case "冬天", "冬季":
			startMonth, endMonth = 12, 2
			startDay, endDay = 1, 28
			hasSeason = true
		}

		// 月份
		if m == "上个月" {
			lm := now.AddDate(0, -1, 0)
			startYear, endYear = lm.Year(), lm.Year()
			startMonth, endMonth = lm.Month(), lm.Month()
			startDay = 1
			endDay = lastDayOfMonth(startYear, startMonth)
			hasMonth = true
		} else if m == "本月" || m == "这个月" {
			startYear, endYear = year, year
			startMonth, endMonth = now.Month(), now.Month()
			startDay = 1
			endDay = lastDayOfMonth(startYear, startMonth)
			hasMonth = true
		}

		// 周
		if m == "上周" {
			weekday := int(now.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			lastMon := now.AddDate(0, 0, -weekday-6)
			lastSun := now.AddDate(0, 0, -weekday)
			return lastMon.Format("2006-01-02"), lastSun.Format("2006-01-02"), true
		} else if m == "本周" {
			weekday := int(now.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			thisMon := now.AddDate(0, 0, -weekday+1)
			return thisMon.Format("2006-01-02"), now.Format("2006-01-02"), true
		}

		// 节日/特定日期（简化处理）
		switch m {
		case "元旦", "新年":
			if !hasYear {
				startYear, endYear = year, year
			}
			startMonth, endMonth = 1, 1
			startDay, endDay = 1, 1
			hasDay = true
		case "春节":
			if !hasYear {
				startYear, endYear = year, year
			}
			startMonth, endMonth = 1, 2
			startDay, endDay = 1, 28
			hasDay = true
		}
	}

	// 处理季节跨年（冬天）
	if hasSeason && startMonth == 12 && endMonth == 2 {
		fromDate := time.Date(startYear, startMonth, startDay, 0, 0, 0, 0, time.Local)
		// 结束日期是次年
		toDate := time.Date(endYear+1, endMonth, endDay, 0, 0, 0, 0, time.Local)
		return fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"), true
	}

	// 如果没有年份但有季节/月份，默认使用去年（因为照片通常是过去拍的）
	if !hasYear && (hasSeason || hasMonth || hasDay) {
		startYear = year - 1
		endYear = year - 1
	}

	fromDate := time.Date(startYear, startMonth, startDay, 0, 0, 0, 0, time.Local)
	toDate := time.Date(endYear, endMonth, endDay, 0, 0, 0, 0, time.Local)

	// 修正最后一天（比如2月）
	if hasSeason || !hasDay {
		endDay = lastDayOfMonth(endYear, endMonth)
		toDate = time.Date(endYear, endMonth, endDay, 0, 0, 0, 0, time.Local)
	}

	return fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"), true
}

func lastDayOfMonth(year int, month time.Month) int {
	next := time.Date(year, month+1, 1, 0, 0, 0, 0, time.Local)
	return next.AddDate(0, 0, -1).Day()
}
