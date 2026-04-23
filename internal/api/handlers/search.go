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
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	limitStr := c.DefaultQuery("limit", "20")
	_, _ = strconv.Atoi(limitStr) // limit 保留兼容，实际用 pageSize

	// 获取过滤参数
	period := c.DefaultQuery("period", "all")
	tagsParam := c.Query("tags")
	dir := c.Query("dir")
	var filterTags []string
	if tagsParam != "" {
		filterTags = strings.Split(tagsParam, ",")
	}

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
		// 暂不应用 LLM 推断的标签过滤（L1 标签库尚未建立）
		// if parsed.Tags != "" {
		// 	for _, t := range strings.Split(parsed.Tags, ",") {
		// 		t = strings.TrimSpace(t)
		// 		if t != "" {
		// 			filterTags = append(filterTags, t)
		// 		}
		// 	}
		// }
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
			hasTag := false
			for _, ft := range filterTags {
				for _, t := range p.Tags {
					if strings.TrimSpace(ft) == t {
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
