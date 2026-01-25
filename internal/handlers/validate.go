package handlers
import "net/http"
import "mime"

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