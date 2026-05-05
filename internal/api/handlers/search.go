package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"photo-search-engine/internal/indexer"
	"photo-search-engine/internal/query"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SearchHandler struct {
	indexer   *indexer.Index
	logger    *zap.Logger
	llmClient query.LLMClient
	l2Engine  *query.L2Engine
}

type photoWithL2 struct {
	indexer.Photo
	L2 query.L2Result `json:"l2,omitempty"`
}

func NewSearchHandler(idx *indexer.Index, logger *zap.Logger) *SearchHandler {
	l2e, _ := query.NewL2Engine("")
	llm, _ := query.NewLLMClientFromEnv()
	return &SearchHandler{indexer: idx, logger: logger, llmClient: llm, l2Engine: l2e}
}

func NewSearchHandlerWithLLM(idx *indexer.Index, logger *zap.Logger, llmClient query.LLMClient) *SearchHandler {
	l2e, _ := query.NewL2Engine("")
	return &SearchHandler{indexer: idx, logger: logger, llmClient: llmClient, l2Engine: l2e}
}

func (h *SearchHandler) HandleSearch(c *gin.Context) {
	q := c.Query("q")
	tagsParam := c.Query("tags")
	if q == "" && tagsParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "'q' or 'tags' parameter is required"})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "40")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 40
	}

	limitStr := c.DefaultQuery("limit", "40")
	_, _ = strconv.Atoi(limitStr) // limit 保留兼容，实际用 pageSize

	// 获取过滤参数
	period := c.DefaultQuery("period", "all")
	dir := c.Query("dir")

	// 计算日期范围
	var dateFrom, dateTo string
	now := time.Now()
	switch period {
	case "week":
		dateFrom = now.AddDate(0, 0, -int(now.Weekday())+1).Format("2006-01-02")
	case "month":
		dateFrom = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).Format("2006-01-02")
	case "year":
		dateFrom = time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.Local).Format("2006-01-02")
	}
	if period != "all" {
		dateTo = now.Format("2006-01-02")
	}

	searchQuery := q
	llmParsed := false
	requireAllTags := false

	// 处理显式 tags 参数
	matchAllParam := c.Query("match_all")
	var filterTags []string
	if tagsParam != "" {
		filterTags = strings.Split(tagsParam, ",")
		if len(filterTags) > 1 && matchAllParam != "false" {
			requireAllTags = true // 多标签默认 AND
		}
	}
	// 尝试 LLM 自然语言解析
	if h.llmClient != nil {
		if parsed, err := query.ParseNaturalLanguage(h.llmClient, q); err == nil && parsed != nil {
		if parsed.Query != "" || len(parsed.TimeMarkers) > 0 {
			searchQuery = parsed.Query
			llmParsed = true
		}
		if dFrom, dTo, ok := query.ResolveDateRange(parsed.TimeMarkers); ok {
			if dFrom != "" {
				dateFrom = dFrom
			}
			if dTo != "" {
				dateTo = dTo
			}
		}
		// 应用 LLM 推断的标签过滤
		if parsed.Tags != "" {
			for _, t := range strings.Split(parsed.Tags, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					filterTags = append(filterTags, t)
				}
			}
			// 复合查询检测（用原始查询 q，不被 LLM 改写）
			requireAllTags = parsed.RequireAllTags || isCompoundQuery(q)
		}
		h.logger.Info("Query parsed by LLM", zap.String("original", q), zap.Any("parsed", parsed), zap.String("dateFrom", dateFrom), zap.String("dateTo", dateTo))
		} else if err != nil {
			h.logger.Warn("LLM query parse failed, fallback to raw query", zap.Error(err))
		}
	} else {
		h.logger.Warn("LLM client not available, skip query parsing")
	}

	var results []indexer.Photo
	var err error
	if searchQuery == "" && llmParsed {
		// 如果 LLM 解析后 query 为空（如"2024年拍的照片"），直接取全部照片
		results = h.indexer.ListAllPhotos()
	} else {
		results, err = h.indexer.Search(searchQuery, 1000)
		if err != nil {
			h.logger.Error("Search failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
			return
		}
	}

	// 过滤结果并附加L2标签
	queryOnlyTime := searchQuery == "" && (dateFrom != "" || dateTo != "")
	allFiltered := make([]photoWithL2, 0)
	for _, p := range results {
		// 目录过滤
		if dir != "" && !strings.HasPrefix(p.Path, dir) {
			continue
		}
		if p.DateTime == "" || p.DateTime == "N/A" {
			// 纯时间查询时，排除无日期照片
			if queryOnlyTime {
				continue
			}
		} else {
			if dateFrom != "" && p.DateTime < dateFrom {
				continue
			}
			if dateTo != "" && p.DateTime > dateTo {
				continue
			}
		}
		if len(filterTags) > 0 {
			if requireAllTags {
				// 硬 AND：复合查询必须匹配所有概念
				matchedAll := true
				for _, ft := range filterTags {
					ft = strings.TrimSpace(ft)
					hasThis := false
					for _, t := range p.Tags {
						if tagMatchesConcept(t, ft) {
							hasThis = true
							break
						}
					}
					if !hasThis {
						matchedAll = false
						break
					}
				}
				if !matchedAll {
					continue
				}
			} else {
				// OR 匹配：简单查询或并列查询，有任一标签即可
				hasTag := false
				for _, ft := range filterTags {
					ft = strings.TrimSpace(ft)
					for _, t := range p.Tags {
						if strings.Contains(t, ft) || strings.Contains(ft, t) {
							hasTag = true
							break
						}
					}
					if hasTag {
						break
					}
				}
				if !hasTag {
					continue
				}
			}
		}
		pw := photoWithL2{Photo: p}
		if h.l2Engine != nil {
			pw.L2 = h.l2Engine.Infer(p.Tags)
		}
		allFiltered = append(allFiltered, pw)
	}



	// 分页
	total := len(allFiltered)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	var filtered []photoWithL2
	if start < total {
		filtered = allFiltered[start:end]
	} else {
		filtered = []photoWithL2{}
	}

	c.JSON(http.StatusOK, gin.H{
		"query":      searchQuery,
		"period":     period,
		"dir":        dir,
		"date_from":  dateFrom,
		"date_to":    dateTo,
		"results":    filtered,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// isCompoundQuery 检测查询是否为复合查询（如"带花的人像"→需要AND匹配）
func isCompoundQuery(q string) bool {
	exactPatterns := []string{"里边有", "旁边有", "前面有", "后面有", "上面有", "下面有", "里面有"}
	for _, p := range exactPatterns {
		if strings.Contains(q, p) {
			return true
		}
	}
	words := []rune(q)
	for i := 0; i < len(words)-2; i++ {
		if (words[i] == '带' || words[i] == '有') && words[i+2] == '的' && words[i+1] != ' ' {
			return true
		}
		if i+2 < len(words) && words[i+1] == '中' && words[i+2] == '的' && words[i] != ' ' {
			return true
		}
		if i+2 < len(words) && words[i+1] == '里' && words[i+2] == '的' && words[i] != ' ' {
			return true
		}
	}
	return false
}

// tagMatchesConcept 标签是否匹配概念关键词（语义模糊匹配）
func tagMatchesConcept(tag string, concept string) bool {
	conceptKeywords := map[string][]string{
		"花":   {"花"},
		"花卉": {"花"},
		"人像": {"人", "肖像", "亲子", "儿童", "情侣", "自拍", "人物", "老人", "单人"},
		"人物": {"人", "肖像", "亲子", "儿童", "情侣", "自拍", "老人", "单人"},
		"人物合影": {"人", "肖像", "亲子", "儿童", "情侣", "自拍", "老人", "单人"},
		"狗":   {"狗", "宠物"},
		"猫":   {"猫", "宠物"},
		"宠物": {"猫", "狗", "宠物", "鸟"},
		"雪":   {"雪"},
		"雪景": {"雪"},
		"海":   {"海", "滨", "滩"},
		"海滨": {"海", "滨", "滩"},
		"日落": {"日落", "日出"},
		"日出": {"日出", "日落"},
		"夜景": {"夜景", "灯光", "霓虹"},
		"城市": {"城市", "街", "楼"},
		"建筑": {"建筑", "古建", "宫", "庙", "楼"},
		"古建筑": {"古建", "宫", "庙", "城", "遗址"},
		"飞机": {"飞机", "航空"},
		"车":   {"车", "汽车", "自行车"},
		"美食": {"美食", "甜点", "蛋糕", "水果", "餐"},
		"动物": {"狗", "猫", "鸟", "野生动", "宠物"},
		"天空": {"天空", "云", "蓝天"},
		"山":   {"山"},
		"树":   {"树", "森林", "植物"},
		"水":   {"水", "湖", "海", "河", "倒影"},
	}
	keywords, ok := conceptKeywords[concept]
	if !ok {
		return strings.Contains(tag, concept) || strings.Contains(concept, tag)
	}
	for _, kw := range keywords {
		if strings.Contains(tag, kw) {
			return true
		}
	}
	return false
}
