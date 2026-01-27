package handlers

import (
    "net/http"
    "taskTracker/internal/logger"
    "taskTracker/internal/service"
    "go.uber.org/zap"
)


func handleBusinessError(w http.ResponseWriter, err error, defaultMessage string) bool {
    if businessErr, ok := err.(*service.BusinessError); ok {
        statusCode := mapBusinessErrorToHTTP(businessErr.Code)
        
        logger.Warn("HTTP: Бизнес-ошибка",
            zap.String("error_code", businessErr.Code),
            zap.Int("http_status", statusCode))
        
        responseWithJSON(w, statusCode, 
            toPayload("error",   businessErr.Code),
            toPayload("message", businessErr.Message),
            toPayload("details", businessErr.Details),
        )
        return true
    }
    return false
}


func mapBusinessErrorToHTTP(code string) int {
    switch code {
    case "NOT_FOUND":
        return http.StatusNotFound
    case "VALIDATION_ERROR":
        return http.StatusBadRequest
    case "ALREADY_ARCHIVED", "NOT_ARCHIVED", "VERSION_CONFLICT":
        return http.StatusConflict
    case "TASK_DELETED", "RESTORE_EXPIRED":
        return http.StatusGone
    case "IN_PROGRESS", "NOT_DELETED":
        return http.StatusConflict
    default:
        return http.StatusBadRequest
    }
}