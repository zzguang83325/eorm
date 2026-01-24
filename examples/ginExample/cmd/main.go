package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/zzguang83325/eorm/examples/ginExample/config"
	"github.com/zzguang83325/eorm/examples/ginExample/internal/service"
)

func main() {
	// 1. 初始化数据库
	dsn := "root:123456@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"
	if err := config.InitDB(dsn); err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}

	// 2. 初始化服务
	svc := service.NewUserService()

	// 3. 注册 HTTP 路由

	// 静态文件服务
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("../web/"))))

	// 主页路由
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "../web/index.html")
			return
		}
		http.NotFound(w, r)
	})

	// API 路由：用户注册（支持 POST 和 GET）
	http.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		var username string
		if r.Method == http.MethodPost {
			// 处理 JSON 请求体
			if r.Header.Get("Content-Type") == "application/json" {
				var req struct {
					Username string `json:"username"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "Invalid JSON", http.StatusBadRequest)
					return
				}
				username = req.Username
			} else {
				// 处理表单数据
				r.ParseForm()
				username = r.FormValue("username")
			}
		} else {
			// GET 方法，从查询参数获取
			username = r.URL.Query().Get("username")
		}

		if username == "" {
			http.Error(w, "Username required", http.StatusBadRequest)
			return
		}

		id, err := svc.Register(username)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"user_id": id,
		})
	})

	http.HandleFunc("/api/checkin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		var userID int64
		if r.Method == http.MethodPost {
			if r.Header.Get("Content-Type") == "application/json" {
				var req struct {
					UserID int64 `json:"user_id"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "Invalid JSON", http.StatusBadRequest)
					return
				}
				userID = req.UserID
			} else {
				r.ParseForm()
				userIDStr := r.FormValue("user_id")
				userID, _ = strconv.ParseInt(userIDStr, 10, 64)
			}
		} else {
			userIdStr := r.URL.Query().Get("user_id")
			userID, _ = strconv.ParseInt(userIdStr, 10, 64)
		}

		err := svc.UserCheckIn(userID)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Checked in successfully, +10 points",
		})
	})

	http.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		username := r.URL.Query().Get("username")
		user, err := svc.GetUserInfo(username)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			})
			return
		}
		if user == nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": "User not found",
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"user":   user,
		})
	})

	// 新增 API：获取所有用户
	http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 获取所有用户（这里需要添加一个获取所有用户的方法）
		users, err := svc.GetAllUsers()
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"users":  users,
		})
	})

	log.Println("Server starting on :18080...")
	log.Println("Try: curl -X POST 'http://localhost:8080/api/register?username=test'")
	if err := http.ListenAndServe(":18080", nil); err != nil {
		log.Fatal(err)
	}
}
