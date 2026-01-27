package handlers

import (
    "net/http"
    "mime"
    "strconv"
    "github.com/google/uuid"
    "taskTracker/internal/logger"
    "go.uber.org/zap"
	"github.com/go-chi/chi/v5"
)

func checkContentType(r *http.Request, target string) bool {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	return mediaType == target
}


func validatePagination(w http.ResponseWriter, r *http.Request) (page, limit int, ok bool) {
    pageStr := r.URL.Query().Get("page")
    limitStr := r.URL.Query().Get("limit")
    
  
    if pageStr == "" {
        pageStr = "1"
    }
    if limitStr == "" {
        limitStr = "50"
    }
    
    page, err := strconv.Atoi(pageStr)
    if err != nil || page <= 0 {
        logger.Warn("HTTP: Неверное значение параметра",
            zap.String("query", "page"),
            zap.String("value", pageStr),
            zap.String("client_ip", r.RemoteAddr))
        responseWithError(w, http.StatusBadRequest, "параметр page должен быть положительным числом")
        return 0, 0, false
    }
    
    limit, err = strconv.Atoi(limitStr)
    if err != nil || limit <= 0 {
        logger.Warn("HTTP: Неверное значение параметра",
            zap.String("query", "limit"),
            zap.String("value", limitStr),
            zap.String("client_ip", r.RemoteAddr))
        responseWithError(w, http.StatusBadRequest, "параметр limit должен быть положительным числом")
        return 0, 0, false
    }
    
    
    if limit > 1000 {
        limit = 1000
    }
    
    return page, limit, true
}


func validateUUID(w http.ResponseWriter, r *http.Request, paramName string) (uuid.UUID, bool) {
    idParam := chi.URLParam(r, paramName)
    if idParam == "" {
        logger.Warn("HTTP: Отсутствует параметр ID",
            zap.String("param", paramName),
            zap.String("client_ip", r.RemoteAddr))
        responseWithError(w, http.StatusBadRequest, "отсутствует идентификатор задачи")
        return uuid.Nil, false
    }
    
    id, err := uuid.Parse(idParam)
    if err != nil {
        logger.Warn("HTTP: Неверный формат UUID",
            zap.String("param", paramName),
            zap.String("value", idParam),
            zap.String("client_ip", r.RemoteAddr))
        responseWithError(w, http.StatusBadRequest, "неверный формат идентификатора задачи")
        return uuid.Nil, false
    }
    
    return id, true
}