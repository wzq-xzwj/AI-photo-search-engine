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
	Query       string   `json:"query"`
	TimeMarkers []string `json:"time_markers"`
	Tags        string   `json:"tags"`
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
	return fmt.Sprintf(`你是一个照片搜索查询解析助手。请将用户的自然语言查询解析为结构化 JSON。

用户查询："%s"

请只输出一个 JSON 对象，包含以下字段：
- query: 清洗后的核心搜索词（用于语义搜索），保留地点、人物、事件、物体等关键信息，去掉时间修饰语。对于语义相近的词，请扩展为更通用的表达以提高搜索命中率。例如："人像"→"人"、"孩童"→"小孩 儿童"、"车辆"→"车 汽车"
- time_markers: 时间关键词数组，如 ["去年","夏天"]。如果没有时间限定则空数组 []
- tags: 可能的标签过滤条件，用逗号分隔，没有则空字符串

语义扩展示例：
- "人像" → {"query":"人","time_markers":[],"tags":"人像,人物,人脸"}
- "孩童在公园玩" → {"query":"小孩 儿童 公园 玩","time_markers":[],"tags":"小孩,公园"}
- "车辆照片" → {"query":"车 汽车","time_markers":[],"tags":"车辆,汽车"}

时间关键词定义：
- "去年" = 上一个自然年
- "今年" = 当前自然年
- "上个月" = 上一个自然月
- "上周" = 上一个自然周（周一至周日）
- "夏天" = 6月1日至8月31日
- "冬天" = 12月1日至次年2月28日
- "春天" = 3月1日至5月31日
- "秋天" = 9月1日至11月30日
- "元旦" = 1月1日
- "春节" = 1月或2月（可只标记"春节"，不用推算具体日期）

示例：
输入："去年夏天在故宫拍的照片"
输出：{"query":"故宫","time_markers":["去年","夏天"],"tags":""}

输入："找我和朋友在餐厅聚餐的照片"
输出：{"query":"餐厅 聚餐 朋友","time_markers":[],"tags":"餐厅,聚餐"}

输入："2025年元旦在北京的照片"
输出：{"query":"北京","time_markers":["2025年","元旦"],"tags":""}

输入："2024年拍的照片"
输出：{"query":"","time_markers":["2024年"],"tags":""}

输入："上个月在香港的照片"
输出：{"query":"香港","time_markers":["上个月"],"tags":""}

只输出 JSON，不要任何解释。`, query)
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
