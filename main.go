package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Token管理
type TokenManager struct {
	tokens map[string]time.Time
	mu     sync.RWMutex
}

var tokenManager = &TokenManager{
	tokens: make(map[string]time.Time),
}

// 上传目录路径
var uploadDir string

// 生成随机token
func generateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)
	return token, nil
}

// 添加token
func (tm *TokenManager) AddToken(token string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.tokens[token] = time.Now().Add(1 * time.Hour) // Token 1小时有效
}

// 验证token
func (tm *TokenManager) ValidateToken(token string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	expiry, exists := tm.tokens[token]
	if !exists {
		return false
	}
	
	if time.Now().After(expiry) {
		delete(tm.tokens, token)
		return false
	}
	
	return true
}

// 清理过期tokens
func (tm *TokenManager) cleanupExpiredTokens() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	now := time.Now()
	for token, expiry := range tm.tokens {
		if now.After(expiry) {
			delete(tm.tokens, token)
		}
	}
}

// 定期清理过期tokens
func (tm *TokenManager) StartCleanupRoutine() {
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for range ticker.C {
			tm.cleanupExpiredTokens()
		}
	}()
}

// 创建token处理器
func createTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token, err := generateToken()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	tokenManager.AddToken(token)
	fmt.Fprintf(w, "%s", token)
	log.Printf("Generated token: %s", token)
}

// 上传文件处理器
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 验证token
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	
	if token == "" || !tokenManager.ValidateToken(token) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 解析multipart表单，设置最大内存为32MB
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// 获取文件句柄
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 创建文件保存路径
	filename := handler.Filename
	// 防止路径遍历攻击
	filename = filepath.Base(filename)
	
	// 确保上传目录存在
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}
	
	// 创建目标文件
	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		http.Error(w, "Unable to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// 将上传的文件拷贝到目标文件
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File uploaded successfully: %s\n", filename)
	log.Printf("Uploaded file: %s to %s", filename, uploadDir)
}

// 下载文件处理器
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 验证token
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	
	if token == "" || !tokenManager.ValidateToken(token) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 获取文件名
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	filename := pathParts[2]
	
	// 防止路径遍历攻击
	filename = filepath.Base(filename)
	filePath := filepath.Join(uploadDir, filename)
	
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// 设置响应头
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Expires", "0")
	w.Header().Set("Cache-Control", "must-revalidate")
	w.Header().Set("Pragma", "public")

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Unable to open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Unable to get file info", http.StatusInternalServerError)
		return
	}

	// 设置Content-Length
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// 将文件写入响应
	_, err = io.Copy(w, file)
	if err != nil {
		log.Printf("Error serving file %s: %v", filename, err)
		return
	}
	
	log.Printf("Downloaded file: %s", filename)
}

func main() {
	var (
		serverAddr = flag.String("server", "localhost", "Server address")
		port       = flag.String("port", "8080", "Server port")
		uploadPath = flag.String("upload-dir", "uploads", "Upload directory path")
	)
	flag.Parse()

	// 设置上传目录
	uploadDir = *uploadPath

	// 启动定期清理过期tokens的任务
	tokenManager.StartCleanupRoutine()

	// 注册路由处理器
	http.HandleFunc("/token", createTokenHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/download/", downloadHandler)

	// 默认首页显示帮助信息
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		helpText := `
<!DOCTYPE html>
<html>
<head>
    <title>File Server</title>
</head>
<body>
    <h1>File Server with Token Authentication</h1>
    <p>This server supports uploading and downloading large files with token authentication.</p>
    
    <h2>Usage:</h2>
    <ol>
        <li><strong>Generate Token:</strong><br>
            <code>curl -X POST http://%s:%s/token</code>
        </li>
        <li><strong>Upload File:</strong><br>
            <code>curl -X POST -F "file=@yourfile.txt" http://%s:%s/upload?token=TOKEN</code>
        </li>
        <li><strong>Download File:</strong><br>
            <code>curl -X GET http://%s:%s/download/filename?token=TOKEN -o downloaded_file.txt</code>
        </li>
    </ol>
    
    <p>Note: TOKEN is the token received from the first step.</p>
</body>
</html>
`
		fmt.Fprintf(w, helpText, *serverAddr, *port, *serverAddr, *port, *serverAddr, *port)
	})

	addr := fmt.Sprintf("%s:%s", *serverAddr, *port)
	log.Printf("Starting server on %s", addr)
	log.Printf("Upload directory: %s", uploadDir)
	log.Fatal(http.ListenAndServe(addr, nil))
}