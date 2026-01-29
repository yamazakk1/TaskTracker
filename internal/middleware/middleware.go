package middleware

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync"
	"taskTracker/internal/logger"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const RequestIdKey contextKey = "request_id"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestId := r.Header.Get("X-Request-ID")
		if requestId == "" {
			requestId = uuid.New().String()
		}

		w.Header().Set("X-Request-ID", requestId)

		ctx := context.WithValue(r.Context(), RequestIdKey, requestId)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

type loggingWriter struct {
	http.ResponseWriter
	status      int
	size        int
	wroteHeader bool
}

func (lw *loggingWriter) WriteHeader(code int) {
	if !lw.wroteHeader {
		lw.status = code
		lw.wroteHeader = true
		lw.ResponseWriter.WriteHeader(code)
	}

}

func (lw *loggingWriter) Write(b []byte) (int, error) {
	if !lw.wroteHeader {
		lw.WriteHeader(http.StatusOK)
	}

	n, err := lw.ResponseWriter.Write(b)
	lw.size += n
	return n, err
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requesId := GetRequestID(r.Context())

		logger.Info(
			"HTTP_IN: Начало зароса",
			zap.String("request_id", requesId),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("query", r.URL.RawQuery),
			zap.String("clietn_ip", r.RemoteAddr),
		)

		lw := &loggingWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
			size:           0,
			wroteHeader:    false,
		}
		next.ServeHTTP(lw, r)

		logLevel := zap.InfoLevel
		if lw.status >= 400 && lw.status < 500 {
			logLevel = zap.WarnLevel
		} else if lw.status >= 500 {
			logLevel = zap.ErrorLevel
		}
		logger.Log(
			logLevel,
			"HTTP_OUT: Завершение запроса",
			zap.String("request_id", requesId),
			zap.Int("status", lw.status),
			zap.Int("bytes_written", lw.size),
			zap.Duration("ms", time.Since(start)),
		)

	})
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIdKey).(string); ok {
		return id
	}
	return ""
}

// func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
// 	return func(next http.Handler) http.Handler {
// 		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			requestId := GetRequestID(r.Context())

// 			ctx, cancel := context.WithTimeout(r.Context(), timeout)
// 			defer cancel()

// 			r = r.WithContext(ctx)

// 			done := make(chan struct{}, 1)

// 			go func() {
// 				next.ServeHTTP(w, r)
// 				close(done)
// 			}()

// 			select {
// 			case <-done:
// 				return
// 			case <-ctx.Done():
// 				if ctx.Err() == context.DeadlineExceeded {
// 					logger.Warn(
// 						"HTTP: таймаут запроса",
// 						zap.String("request_id", requestId),
// 						zap.String("method", r.Method),
// 						zap.String("path", r.URL.Path),
// 						zap.String("client_ip", r.RemoteAddr),
// 						zap.Duration("ms", timeout),
// 					)
// 					w.WriteHeader(http.StatusGatewayTimeout)
// 					w.Header().Set("Content-Type", "application/json")

// 					w.Write([]byte(`{
//                         "error": "request timeout",
//                         "request_id": "` + requestId + `",
//                         "message": "the request took too long to process"
//                     }`))

// 					if hijacker, ok := w.(http.Hijacker); ok {
// 						if conn, _, err := hijacker.Hijack(); err == nil {
// 							conn.Close()
// 						}
// 					}
// 				}
// 			}
// 		})
// 	}
// }

type clientInfo struct {
	count   int
	resetAt time.Time
}

func RateLimit(rpm int) func(http.Handler) http.Handler {
	clients := make(map[string]*clientInfo)
	var mtx sync.Mutex
	window := time.Minute

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIp(r)
			now := time.Now()

			mtx.Lock()

			// Всегда работаем с одной переменной info
			info, exists := clients[ip]

			if !exists {
				// Создаем новую запись
				info = &clientInfo{
					count:   1,
					resetAt: now.Add(window),
				}
				clients[ip] = info
			} else if now.After(info.resetAt) {
				// Сброс счетчика
				info.count = 1
				info.resetAt = now.Add(window)
			} else {
				// Проверяем лимит
				if info.count >= rpm {
					mtx.Unlock() // разблокируем перед возвратом

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusTooManyRequests)

					json.NewEncoder(w).Encode(map[string]any{
						"error":       "rate_limit_exceeded",
						"message":     "Слишком много запросов. Попробуйте позже.",
						"retry_after": int(info.resetAt.Sub(now).Seconds()),
						"request_id":  GetRequestID(r.Context()),
					})
					return
				}

				// Увеличиваем счетчик
				info.count++
			}

			// Сохраняем значения ДО разблокировки
			remaining := rpm - info.count
			resetUnix := info.resetAt.Unix()

			mtx.Unlock() // разблокируем здесь

			// Устанавливаем заголовки
			if remaining < 0 {
				remaining = 0
			}
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rpm))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetUnix, 10))

			// Вызываем следующий handler
			next.ServeHTTP(w, r)
		})
	}
}

func getIp(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
