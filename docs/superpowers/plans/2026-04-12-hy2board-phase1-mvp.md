# hy2board Phase 1 (MVP) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a working admin panel + API that manages Hysteria 2 nodes, users, and generates subscription links for Surge/Clash/NekoBox.

**Architecture:** Go backend (Gin) with embedded React frontend (Vite + Tailwind). SQLite for storage. Communicates with remote Hy2 servers via their Traffic Stats API. Admin manages nodes and users through a clean Vercel-style web UI. Users receive a subscription URL that returns formatted configs for their client app.

**Tech Stack:** Go 1.21+, Gin, GORM, SQLite, React 18, Vite, Tailwind CSS, shadcn/ui, Docker

---

## File Structure

```
hy2board/
├── main.go                          # Entry point, starts server
├── go.mod / go.sum
├── Dockerfile
├── docker-compose.yaml
├── config.yaml                      # App config (listen port, admin creds, JWT secret)
│
├── internal/
│   ├── config/
│   │   └── config.go                # Load config.yaml
│   ├── database/
│   │   └── database.go             # GORM init, auto-migrate
│   ├── model/
│   │   ├── node.go                  # Node model (name, host, port, password, obfs, tls)
│   │   ├── user.go                  # User model (username, password hash, traffic limit, expiry)
│   │   └── subscription.go         # Subscription model (user_id, token, format)
│   ├── handler/
│   │   ├── auth.go                  # POST /api/admin/login
│   │   ├── node.go                  # CRUD /api/admin/nodes
│   │   ├── user.go                  # CRUD /api/admin/users
│   │   ├── stats.go                 # GET /api/admin/stats (dashboard data)
│   │   └── subscribe.go            # GET /api/sub/:token (subscription endpoint)
│   ├── middleware/
│   │   └── auth.go                  # JWT auth middleware
│   ├── service/
│   │   ├── node_health.go          # Background node health checker
│   │   ├── traffic.go              # Query Hy2 Traffic Stats API
│   │   └── subscription.go         # Generate Surge/Clash/NekoBox configs
│   └── util/
│       └── jwt.go                   # JWT token helpers
│
├── web/                             # React frontend (Vite)
│   ├── package.json
│   ├── vite.config.ts
│   ├── tailwind.config.ts
│   ├── index.html
│   └── src/
│       ├── main.tsx
│       ├── App.tsx                   # Router
│       ├── api/
│       │   └── client.ts            # Axios instance + auth interceptor
│       ├── pages/
│       │   ├── Login.tsx
│       │   ├── Dashboard.tsx        # Overview stats
│       │   ├── Nodes.tsx            # Node list + add/edit
│       │   ├── Users.tsx            # User list + add/edit
│       │   └── UserDetail.tsx       # Single user: subscription link, traffic
│       ├── components/
│       │   ├── Layout.tsx           # Sidebar + header
│       │   ├── NodeCard.tsx
│       │   ├── UserTable.tsx
│       │   ├── StatsCard.tsx
│       │   └── Modal.tsx
│       └── lib/
│           └── utils.ts
│
└── data/                            # SQLite database (gitignored)
    └── hy2board.db
```

---

### Task 1: Project Init + Config

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `config.yaml`
- Create: `internal/config/config.go`
- Create: `.gitignore`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/a1-6/importantfile/opensource/hy2board
go mod init github.com/ludandaye/hy2board
```

- [ ] **Step 2: Create config.yaml**

```yaml
# hy2board configuration
server:
  listen: ":8080"

admin:
  username: "admin"
  password: "changeme"

jwt:
  secret: "change-this-to-random-string"
  expiry: "24h"

database:
  path: "./data/hy2board.db"
```

- [ ] **Step 3: Write config loader**

Create `internal/config/config.go`:

```go
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

type AdminConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type JWTConfig struct {
	Secret string `yaml:"secret"`
	Expiry string `yaml:"expiry"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Admin    AdminConfig    `yaml:"admin"`
	JWT      JWTConfig      `yaml:"jwt"`
	Database DatabaseConfig `yaml:"database"`
}

func (c *Config) JWTExpiry() time.Duration {
	d, err := time.ParseDuration(c.Expiry)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}

var C Config

func Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, &C); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Create main.go skeleton**

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ludandaye/hy2board/internal/config"
)

func main() {
	cfgPath := "config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	if err := config.Load(cfgPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("hy2board starting on %s\n", config.C.Server.Listen)
}
```

- [ ] **Step 5: Create .gitignore**

```
data/
*.db
web/node_modules/
web/dist/
hy2board
```

- [ ] **Step 6: Build and run**

```bash
go mod tidy
go run main.go
```

Expected: `hy2board starting on :8080`

- [ ] **Step 7: Commit**

```bash
git init
git add .
git commit -m "feat: project init with config loader"
```

---

### Task 2: Database + Models

**Files:**
- Create: `internal/database/database.go`
- Create: `internal/model/node.go`
- Create: `internal/model/user.go`

- [ ] **Step 1: Install dependencies**

```bash
go get gorm.io/gorm
go get gorm.io/driver/sqlite
go get github.com/gin-gonic/gin
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto
```

- [ ] **Step 2: Create Node model**

Create `internal/model/node.go`:

```go
package model

import "gorm.io/gorm"

type Node struct {
	gorm.Model
	Name         string `gorm:"not null" json:"name"`
	Host         string `gorm:"not null" json:"host"`
	Port         int    `gorm:"not null" json:"port"`
	Password     string `gorm:"not null" json:"password"`
	SNI          string `json:"sni"`
	Insecure     bool   `json:"insecure"`
	ObfsType     string `json:"obfs_type"`
	ObfsPassword string `json:"obfs_password"`
	TrafficAPI   string `json:"traffic_api"`
	TrafficSecret string `json:"traffic_secret"`
	Healthy      bool   `gorm:"default:true" json:"healthy"`
	SortOrder    int    `gorm:"default:0" json:"sort_order"`
}
```

- [ ] **Step 3: Create User model**

Create `internal/model/user.go`:

```go
package model

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username      string    `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash  string    `gorm:"-" json:"-"`
	Hy2Password   string    `gorm:"not null" json:"hy2_password"`
	SubToken      string    `gorm:"uniqueIndex;not null" json:"sub_token"`
	TrafficLimit  int64     `gorm:"default:0" json:"traffic_limit"`
	TrafficUsed   int64     `gorm:"default:0" json:"traffic_used"`
	ExpiresAt     time.Time `json:"expires_at"`
	Enabled       bool      `gorm:"default:true" json:"enabled"`
	NodeIDs       string    `gorm:"default:'all'" json:"node_ids"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.SubToken == "" {
		b := make([]byte, 16)
		rand.Read(b)
		u.SubToken = hex.EncodeToString(b)
	}
	if u.Hy2Password == "" {
		b := make([]byte, 16)
		rand.Read(b)
		u.Hy2Password = u.Username + ":" + hex.EncodeToString(b)
	}
	return nil
}

func (u *User) IsExpired() bool {
	return !u.ExpiresAt.IsZero() && time.Now().After(u.ExpiresAt)
}

func (u *User) TrafficExceeded() bool {
	return u.TrafficLimit > 0 && u.TrafficUsed >= u.TrafficLimit
}

func (u *User) IsActive() bool {
	return u.Enabled && !u.IsExpired() && !u.TrafficExceeded()
}
```

- [ ] **Step 4: Create database init**

Create `internal/database/database.go`:

```go
package database

import (
	"os"
	"path/filepath"

	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}

	return DB.AutoMigrate(&model.Node{}, &model.User{})
}
```

- [ ] **Step 5: Wire up in main.go**

Update `main.go`:

```go
package main

import (
	"log"
	"os"

	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/database"
)

func main() {
	cfgPath := "config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	if err := config.Load(cfgPath); err != nil {
		log.Fatalf("Config: %v", err)
	}

	if err := database.Init(config.C.Database.Path); err != nil {
		log.Fatalf("Database: %v", err)
	}

	log.Printf("hy2board ready, database at %s", config.C.Database.Path)
}
```

- [ ] **Step 6: Build and run**

```bash
go run main.go
```

Expected: `hy2board ready, database at ./data/hy2board.db`

- [ ] **Step 7: Commit**

```bash
git add .
git commit -m "feat: database models for nodes and users"
```

---

### Task 3: JWT Auth + Admin Login API

**Files:**
- Create: `internal/util/jwt.go`
- Create: `internal/middleware/auth.go`
- Create: `internal/handler/auth.go`

- [ ] **Step 1: Create JWT utility**

Create `internal/util/jwt.go`:

```go
package util

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ludandaye/hy2board/internal/config"
)

func GenerateToken(username string) (string, error) {
	expiry, _ := time.ParseDuration(config.C.JWT.Expiry)
	if expiry == 0 {
		expiry = 24 * time.Hour
	}

	claims := jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(expiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.C.JWT.Secret))
}

func ParseToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.C.JWT.Secret), nil
	})
	if err != nil {
		return "", err
	}
	claims := token.Claims.(jwt.MapClaims)
	return claims["sub"].(string), nil
}
```

- [ ] **Step 2: Create auth middleware**

Create `internal/middleware/auth.go`:

```go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/util"
)

func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		username, err := util.ParseToken(strings.TrimPrefix(auth, "Bearer "))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("admin", username)
		c.Next()
	}
}
```

- [ ] **Step 3: Create auth handler**

Create `internal/handler/auth.go`:

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/util"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Username != config.C.Admin.Username || req.Password != config.C.Admin.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wrong credentials"})
		return
	}

	token, err := util.GenerateToken(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
```

- [ ] **Step 4: Commit**

```bash
git add .
git commit -m "feat: JWT auth + admin login endpoint"
```

---

### Task 4: Node CRUD API

**Files:**
- Create: `internal/handler/node.go`

- [ ] **Step 1: Create node handler**

Create `internal/handler/node.go`:

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

func ListNodes(c *gin.Context) {
	var nodes []model.Node
	database.DB.Order("sort_order asc").Find(&nodes)
	c.JSON(http.StatusOK, nodes)
}

func CreateNode(c *gin.Context) {
	var node model.Node
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.Create(&node)
	c.JSON(http.StatusCreated, node)
}

func UpdateNode(c *gin.Context) {
	var node model.Node
	if database.DB.First(&node, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}

	var input model.Node
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database.DB.Model(&node).Updates(input)
	c.JSON(http.StatusOK, node)
}

func DeleteNode(c *gin.Context) {
	if database.DB.Delete(&model.Node{}, c.Param("id")).RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
```

- [ ] **Step 2: Commit**

```bash
git add .
git commit -m "feat: node CRUD API"
```

---

### Task 5: User CRUD API

**Files:**
- Create: `internal/handler/user.go`

- [ ] **Step 1: Create user handler**

Create `internal/handler/user.go`:

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

func ListUsers(c *gin.Context) {
	var users []model.User
	database.DB.Find(&users)
	c.JSON(http.StatusOK, users)
}

func GetUser(c *gin.Context) {
	var user model.User
	if database.DB.First(&user, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func CreateUser(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.Create(&user)
	c.JSON(http.StatusCreated, user)
}

func UpdateUser(c *gin.Context) {
	var user model.User
	if database.DB.First(&user, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	var input model.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database.DB.Model(&user).Updates(input)
	c.JSON(http.StatusOK, user)
}

func DeleteUser(c *gin.Context) {
	if database.DB.Delete(&model.User{}, c.Param("id")).RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func ResetSubToken(c *gin.Context) {
	var user model.User
	if database.DB.First(&user, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	user.SubToken = ""
	database.DB.Save(&user)
	c.JSON(http.StatusOK, user)
}
```

- [ ] **Step 2: Commit**

```bash
git add .
git commit -m "feat: user CRUD API"
```

---

### Task 6: Subscription Generator

**Files:**
- Create: `internal/service/subscription.go`
- Create: `internal/handler/subscribe.go`

- [ ] **Step 1: Create subscription service**

Create `internal/service/subscription.go`:

```go
package service

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ludandaye/hy2board/internal/model"
)

func GenerateSurge(user model.User, nodes []model.Node) string {
	var lines []string
	lines = append(lines, "[Proxy]")

	for _, n := range nodes {
		proxy := fmt.Sprintf("%s = hysteria2, %s, %d, password=%s, skip-cert-verify=%t, sni=%s",
			n.Name, n.Host, n.Port, user.Hy2Password, n.Insecure, n.SNI)
		lines = append(lines, proxy)
	}

	names := make([]string, len(nodes))
	for i, n := range nodes {
		names[i] = n.Name
	}

	lines = append(lines, "")
	lines = append(lines, "[Proxy Group]")
	lines = append(lines, fmt.Sprintf("Auto = url-test, %s, url=http://www.gstatic.com/generate_204, interval=300", strings.Join(names, ", ")))

	return strings.Join(lines, "\n")
}

func GenerateClash(user model.User, nodes []model.Node) string {
	var lines []string
	lines = append(lines, "proxies:")

	for _, n := range nodes {
		lines = append(lines, fmt.Sprintf("  - name: \"%s\"", n.Name))
		lines = append(lines, "    type: hysteria2")
		lines = append(lines, fmt.Sprintf("    server: %s", n.Host))
		lines = append(lines, fmt.Sprintf("    port: %d", n.Port))
		lines = append(lines, fmt.Sprintf("    password: \"%s\"", user.Hy2Password))
		lines = append(lines, fmt.Sprintf("    sni: %s", n.SNI))
		lines = append(lines, fmt.Sprintf("    skip-cert-verify: %t", n.Insecure))
		if n.ObfsType != "" {
			lines = append(lines, fmt.Sprintf("    obfs: %s", n.ObfsType))
			lines = append(lines, fmt.Sprintf("    obfs-password: %s", n.ObfsPassword))
		}
		lines = append(lines, "")
	}

	names := make([]string, len(nodes))
	for i, n := range nodes {
		names[i] = n.Name
	}

	lines = append(lines, "proxy-groups:")
	lines = append(lines, "  - name: Auto")
	lines = append(lines, "    type: url-test")
	lines = append(lines, "    proxies:")
	for _, name := range names {
		lines = append(lines, fmt.Sprintf("      - \"%s\"", name))
	}
	lines = append(lines, "    url: http://www.gstatic.com/generate_204")
	lines = append(lines, "    interval: 300")

	return strings.Join(lines, "\n")
}

func GenerateURI(user model.User, nodes []model.Node) string {
	var uris []string
	for _, n := range nodes {
		u := url.URL{
			Scheme: "hysteria2",
			User:   url.User(user.Hy2Password),
			Host:   fmt.Sprintf("%s:%d", n.Host, n.Port),
		}
		q := u.Query()
		q.Set("sni", n.SNI)
		if n.Insecure {
			q.Set("insecure", "1")
		}
		if n.ObfsType != "" {
			q.Set("obfs", n.ObfsType)
			q.Set("obfs-password", n.ObfsPassword)
		}
		u.RawQuery = q.Encode()
		u.Fragment = n.Name
		uris = append(uris, u.String())
	}
	return strings.Join(uris, "\n")
}
```

- [ ] **Step 2: Create subscribe handler**

Create `internal/handler/subscribe.go`:

```go
package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/service"
)

func Subscribe(c *gin.Context) {
	token := c.Param("token")

	var user model.User
	if database.DB.Where("sub_token = ?", token).First(&user).Error != nil {
		c.String(http.StatusNotFound, "invalid subscription")
		return
	}

	if !user.IsActive() {
		c.String(http.StatusForbidden, "subscription expired or disabled")
		return
	}

	var nodes []model.Node
	database.DB.Where("healthy = ?", true).Order("sort_order asc").Find(&nodes)

	if len(nodes) == 0 {
		c.String(http.StatusServiceUnavailable, "no available nodes")
		return
	}

	// Detect client type from User-Agent or query param
	format := c.DefaultQuery("format", "")
	if format == "" {
		ua := strings.ToLower(c.GetHeader("User-Agent"))
		switch {
		case strings.Contains(ua, "surge"):
			format = "surge"
		case strings.Contains(ua, "clash"), strings.Contains(ua, "mihomo"):
			format = "clash"
		default:
			format = "uri"
		}
	}

	var content string
	switch format {
	case "surge":
		content = service.GenerateSurge(user, nodes)
		c.Header("Content-Type", "text/plain; charset=utf-8")
	case "clash":
		content = service.GenerateClash(user, nodes)
		c.Header("Content-Type", "text/yaml; charset=utf-8")
	default:
		content = service.GenerateURI(user, nodes)
		c.Header("Content-Type", "text/plain; charset=utf-8")
	}

	c.Header("Content-Disposition", "attachment; filename=hy2board.conf")
	c.String(http.StatusOK, content)
}
```

- [ ] **Step 3: Commit**

```bash
git add .
git commit -m "feat: subscription generator (Surge/Clash/URI)"
```

---

### Task 7: Node Health Checker + Stats API

**Files:**
- Create: `internal/service/node_health.go`
- Create: `internal/service/traffic.go`
- Create: `internal/handler/stats.go`

- [ ] **Step 1: Create node health checker**

Create `internal/service/node_health.go`:

```go
package service

import (
	"log"
	"net"
	"time"

	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

func StartHealthChecker(interval time.Duration) {
	go func() {
		for {
			checkAllNodes()
			time.Sleep(interval)
		}
	}()
}

func checkAllNodes() {
	var nodes []model.Node
	database.DB.Find(&nodes)

	for _, node := range nodes {
		healthy := probeNode(node)
		database.DB.Model(&node).Update("healthy", healthy)
		if !healthy {
			log.Printf("Node %s (%s:%d) is DOWN", node.Name, node.Host, node.Port)
		}
	}
}

func probeNode(node model.Node) bool {
	addr := net.JoinHostPort(node.Host, fmt.Sprintf("%d", node.Port))
	conn, err := net.DialTimeout("udp", addr, 3*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Send QUIC probe packet
	probe := buildQUICProbe()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	conn.Write(probe)

	buf := make([]byte, 1500)
	_, err = conn.Read(buf)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "connection reset") {
			return true
		}
		return false
	}
	return true
}

func buildQUICProbe() []byte {
	dcid := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	scid := []byte{0x11, 0x12, 0x13, 0x14}
	pkt := make([]byte, 0, 1200)
	pkt = append(pkt, 0xC0)
	pkt = append(pkt, 0x00, 0x00, 0x00, 0x00)
	pkt = append(pkt, byte(len(dcid)))
	pkt = append(pkt, dcid...)
	pkt = append(pkt, byte(len(scid)))
	pkt = append(pkt, scid...)
	padding := make([]byte, 1200-len(pkt))
	pkt = append(pkt, padding...)
	return pkt
}
```

- [ ] **Step 2: Create traffic service**

Create `internal/service/traffic.go`:

```go
package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ludandaye/hy2board/internal/model"
)

type TrafficData struct {
	TX int64 `json:"tx"`
	RX int64 `json:"rx"`
}

func GetNodeTraffic(node model.Node) (map[string]TrafficData, error) {
	if node.TrafficAPI == "" {
		return nil, fmt.Errorf("no traffic API configured")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", node.TrafficAPI+"/traffic", nil)
	if err != nil {
		return nil, err
	}
	if node.TrafficSecret != "" {
		req.Header.Set("Authorization", node.TrafficSecret)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data map[string]TrafficData
	json.Unmarshal(body, &data)
	return data, nil
}

func GetNodeOnline(node model.Node) (int, error) {
	if node.TrafficAPI == "" {
		return 0, fmt.Errorf("no traffic API configured")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", node.TrafficAPI+"/online", nil)
	if err != nil {
		return 0, err
	}
	if node.TrafficSecret != "" {
		req.Header.Set("Authorization", node.TrafficSecret)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return len(result), nil
}
```

- [ ] **Step 3: Create stats handler**

Create `internal/handler/stats.go`:

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

func GetStats(c *gin.Context) {
	var totalUsers int64
	var activeUsers int64
	var totalNodes int64
	var healthyNodes int64

	database.DB.Model(&model.User{}).Count(&totalUsers)
	database.DB.Model(&model.User{}).Where("enabled = ?", true).Count(&activeUsers)
	database.DB.Model(&model.Node{}).Count(&totalNodes)
	database.DB.Model(&model.Node{}).Where("healthy = ?", true).Count(&healthyNodes)

	c.JSON(http.StatusOK, gin.H{
		"total_users":   totalUsers,
		"active_users":  activeUsers,
		"total_nodes":   totalNodes,
		"healthy_nodes": healthyNodes,
	})
}
```

- [ ] **Step 4: Commit**

```bash
git add .
git commit -m "feat: node health checker + traffic stats + dashboard API"
```

---

### Task 8: Router + Complete Backend

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Complete main.go with all routes**

```go
package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/handler"
	"github.com/ludandaye/hy2board/internal/middleware"
	"github.com/ludandaye/hy2board/internal/service"
)

func main() {
	cfgPath := "config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	if err := config.Load(cfgPath); err != nil {
		log.Fatalf("Config: %v", err)
	}

	if err := database.Init(config.C.Database.Path); err != nil {
		log.Fatalf("Database: %v", err)
	}

	// Start background health checker
	service.StartHealthChecker(30 * time.Second)

	r := gin.Default()

	// Public: subscription endpoint
	r.GET("/api/sub/:token", handler.Subscribe)

	// Auth
	r.POST("/api/admin/login", handler.Login)

	// Admin API (JWT protected)
	admin := r.Group("/api/admin", middleware.AdminAuth())
	{
		admin.GET("/stats", handler.GetStats)

		admin.GET("/nodes", handler.ListNodes)
		admin.POST("/nodes", handler.CreateNode)
		admin.PUT("/nodes/:id", handler.UpdateNode)
		admin.DELETE("/nodes/:id", handler.DeleteNode)

		admin.GET("/users", handler.ListUsers)
		admin.GET("/users/:id", handler.GetUser)
		admin.POST("/users", handler.CreateUser)
		admin.PUT("/users/:id", handler.UpdateUser)
		admin.DELETE("/users/:id", handler.DeleteUser)
		admin.POST("/users/:id/reset-token", handler.ResetSubToken)
	}

	// Serve frontend static files (will be added in frontend task)
	// r.Static("/assets", "./web/dist/assets")
	// r.StaticFile("/", "./web/dist/index.html")
	// r.NoRoute(func(c *gin.Context) {
	// 	c.File("./web/dist/index.html")
	// })

	log.Printf("hy2board starting on %s", config.C.Server.Listen)
	r.Run(config.C.Server.Listen)
}
```

- [ ] **Step 2: Fix missing imports in node_health.go**

Add `"fmt"` and `"strings"` to `internal/service/node_health.go` imports:

```go
import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)
```

- [ ] **Step 3: Build and test**

```bash
go mod tidy
go build -o hy2board
./hy2board
```

Expected: Server starts on `:8080`

Test login:
```bash
curl -X POST http://localhost:8080/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"changeme"}'
```

Expected: `{"token":"eyJ..."}`

- [ ] **Step 4: Commit**

```bash
git add .
git commit -m "feat: complete backend with all routes"
```

---

### Task 9: React Frontend Setup

**Files:**
- Create: `web/` directory with Vite + React + Tailwind + shadcn/ui

- [ ] **Step 1: Create Vite project**

```bash
cd /Users/a1-6/importantfile/opensource/hy2board
npm create vite@latest web -- --template react-ts
cd web
npm install
npm install -D tailwindcss @tailwindcss/vite
npm install axios react-router-dom lucide-react
npm install @radix-ui/react-dialog @radix-ui/react-slot class-variance-authority clsx tailwind-merge
```

- [ ] **Step 2: Configure Tailwind**

Update `web/vite.config.ts`:

```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
```

Update `web/src/index.css`:

```css
@import "tailwindcss";
```

- [ ] **Step 3: Create API client**

Create `web/src/api/client.ts`:

```ts
import axios from 'axios'

const api = axios.create({ baseURL: '/api' })

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export default api
```

- [ ] **Step 4: Create Layout component (Vercel style)**

Create `web/src/components/Layout.tsx`:

```tsx
import { Link, useLocation } from 'react-router-dom'
import { LayoutDashboard, Server, Users, LogOut } from 'lucide-react'

const nav = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/nodes', icon: Server, label: 'Nodes' },
  { to: '/users', icon: Users, label: 'Users' },
]

export default function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation()

  const logout = () => {
    localStorage.removeItem('token')
    window.location.href = '/login'
  }

  return (
    <div className="min-h-screen bg-black text-white flex">
      <aside className="w-56 border-r border-zinc-800 p-4 flex flex-col">
        <h1 className="text-lg font-semibold mb-8 px-2">hy2board</h1>
        <nav className="flex-1 space-y-1">
          {nav.map(({ to, icon: Icon, label }) => (
            <Link
              key={to}
              to={to}
              className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                location.pathname === to
                  ? 'bg-zinc-800 text-white'
                  : 'text-zinc-400 hover:text-white hover:bg-zinc-900'
              }`}
            >
              <Icon size={16} />
              {label}
            </Link>
          ))}
        </nav>
        <button
          onClick={logout}
          className="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-zinc-400 hover:text-white hover:bg-zinc-900 transition-colors"
        >
          <LogOut size={16} />
          Logout
        </button>
      </aside>
      <main className="flex-1 p-8">{children}</main>
    </div>
  )
}
```

- [ ] **Step 5: Create Login page**

Create `web/src/pages/Login.tsx`:

```tsx
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '@/api/client'

export default function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const navigate = useNavigate()

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const { data } = await api.post('/admin/login', { username, password })
      localStorage.setItem('token', data.token)
      navigate('/')
    } catch {
      setError('Wrong credentials')
    }
  }

  return (
    <div className="min-h-screen bg-black flex items-center justify-center">
      <form onSubmit={handleLogin} className="w-80 space-y-4">
        <h1 className="text-2xl font-semibold text-white text-center">hy2board</h1>
        <p className="text-sm text-zinc-500 text-center">Hysteria 2 Management Panel</p>
        {error && <p className="text-red-400 text-sm text-center">{error}</p>}
        <input
          type="text"
          placeholder="Username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          className="w-full px-3 py-2 bg-zinc-900 border border-zinc-800 rounded-lg text-white text-sm focus:outline-none focus:border-zinc-600"
        />
        <input
          type="password"
          placeholder="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="w-full px-3 py-2 bg-zinc-900 border border-zinc-800 rounded-lg text-white text-sm focus:outline-none focus:border-zinc-600"
        />
        <button className="w-full py-2 bg-white text-black rounded-lg text-sm font-medium hover:bg-zinc-200 transition-colors">
          Sign In
        </button>
      </form>
    </div>
  )
}
```

- [ ] **Step 6: Create Dashboard page**

Create `web/src/pages/Dashboard.tsx`:

```tsx
import { useEffect, useState } from 'react'
import api from '@/api/client'
import { Server, Users, Activity, Shield } from 'lucide-react'

interface Stats {
  total_users: number
  active_users: number
  total_nodes: number
  healthy_nodes: number
}

function StatCard({ icon: Icon, label, value, sub }: any) {
  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
      <div className="flex items-center gap-3 mb-4">
        <Icon size={16} className="text-zinc-500" />
        <span className="text-sm text-zinc-500">{label}</span>
      </div>
      <p className="text-3xl font-semibold">{value}</p>
      {sub && <p className="text-sm text-zinc-500 mt-1">{sub}</p>}
    </div>
  )
}

export default function Dashboard() {
  const [stats, setStats] = useState<Stats | null>(null)

  useEffect(() => {
    api.get('/admin/stats').then(({ data }) => setStats(data))
  }, [])

  if (!stats) return <p className="text-zinc-500">Loading...</p>

  return (
    <div>
      <h2 className="text-xl font-semibold mb-6">Dashboard</h2>
      <div className="grid grid-cols-4 gap-4">
        <StatCard icon={Users} label="Total Users" value={stats.total_users} />
        <StatCard icon={Activity} label="Active Users" value={stats.active_users} />
        <StatCard icon={Server} label="Total Nodes" value={stats.total_nodes} />
        <StatCard icon={Shield} label="Healthy Nodes" value={stats.healthy_nodes} />
      </div>
    </div>
  )
}
```

- [ ] **Step 7: Create App router**

Update `web/src/App.tsx`:

```tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import Layout from '@/components/Layout'
import Login from '@/pages/Login'
import Dashboard from '@/pages/Dashboard'
import Nodes from '@/pages/Nodes'
import Users from '@/pages/Users'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem('token')
  if (!token) return <Navigate to="/login" />
  return <Layout>{children}</Layout>
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/" element={<ProtectedRoute><Dashboard /></ProtectedRoute>} />
        <Route path="/nodes" element={<ProtectedRoute><Nodes /></ProtectedRoute>} />
        <Route path="/users" element={<ProtectedRoute><Users /></ProtectedRoute>} />
      </Routes>
    </BrowserRouter>
  )
}
```

- [ ] **Step 8: Create placeholder pages**

Create `web/src/pages/Nodes.tsx`:

```tsx
import { useEffect, useState } from 'react'
import api from '@/api/client'
import { Plus, Trash2, Circle } from 'lucide-react'

export default function Nodes() {
  const [nodes, setNodes] = useState<any[]>([])
  const [showAdd, setShowAdd] = useState(false)
  const [form, setForm] = useState({ name: '', host: '', port: 443, password: '', sni: '', insecure: true })

  const load = () => api.get('/admin/nodes').then(({ data }) => setNodes(data))
  useEffect(() => { load() }, [])

  const add = async () => {
    await api.post('/admin/nodes', form)
    setShowAdd(false)
    setForm({ name: '', host: '', port: 443, password: '', sni: '', insecure: true })
    load()
  }

  const del = async (id: number) => {
    if (!confirm('Delete this node?')) return
    await api.delete(`/admin/nodes/${id}`)
    load()
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold">Nodes</h2>
        <button onClick={() => setShowAdd(!showAdd)} className="flex items-center gap-2 px-3 py-1.5 bg-white text-black rounded-lg text-sm font-medium hover:bg-zinc-200">
          <Plus size={14} /> Add Node
        </button>
      </div>

      {showAdd && (
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 mb-4 grid grid-cols-3 gap-3">
          <input placeholder="Name" value={form.name} onChange={e => setForm({...form, name: e.target.value})} className="px-3 py-2 bg-black border border-zinc-700 rounded-lg text-sm text-white" />
          <input placeholder="Host" value={form.host} onChange={e => setForm({...form, host: e.target.value})} className="px-3 py-2 bg-black border border-zinc-700 rounded-lg text-sm text-white" />
          <input placeholder="Port" type="number" value={form.port} onChange={e => setForm({...form, port: +e.target.value})} className="px-3 py-2 bg-black border border-zinc-700 rounded-lg text-sm text-white" />
          <input placeholder="Password" value={form.password} onChange={e => setForm({...form, password: e.target.value})} className="px-3 py-2 bg-black border border-zinc-700 rounded-lg text-sm text-white" />
          <input placeholder="SNI" value={form.sni} onChange={e => setForm({...form, sni: e.target.value})} className="px-3 py-2 bg-black border border-zinc-700 rounded-lg text-sm text-white" />
          <button onClick={add} className="px-3 py-2 bg-white text-black rounded-lg text-sm font-medium hover:bg-zinc-200">Save</button>
        </div>
      )}

      <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-zinc-800 text-zinc-500">
            <th className="text-left p-4">Status</th><th className="text-left p-4">Name</th><th className="text-left p-4">Host</th><th className="text-left p-4">Port</th><th className="p-4"></th>
          </tr></thead>
          <tbody>
            {nodes.map(n => (
              <tr key={n.ID} className="border-b border-zinc-800/50 hover:bg-zinc-800/30">
                <td className="p-4"><Circle size={8} className={n.healthy ? 'fill-green-500 text-green-500' : 'fill-red-500 text-red-500'} /></td>
                <td className="p-4 font-medium">{n.name}</td>
                <td className="p-4 text-zinc-400">{n.host}</td>
                <td className="p-4 text-zinc-400">{n.port}</td>
                <td className="p-4 text-right"><button onClick={() => del(n.ID)} className="text-zinc-500 hover:text-red-400"><Trash2 size={14} /></button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
```

Create `web/src/pages/Users.tsx`:

```tsx
import { useEffect, useState } from 'react'
import api from '@/api/client'
import { Plus, Trash2, Copy, Check } from 'lucide-react'

export default function Users() {
  const [users, setUsers] = useState<any[]>([])
  const [showAdd, setShowAdd] = useState(false)
  const [form, setForm] = useState({ username: '', traffic_limit: 0, expires_at: '' })
  const [copied, setCopied] = useState<number | null>(null)

  const load = () => api.get('/admin/users').then(({ data }) => setUsers(data))
  useEffect(() => { load() }, [])

  const add = async () => {
    await api.post('/admin/users', {
      ...form,
      traffic_limit: form.traffic_limit * 1024 * 1024 * 1024,
      expires_at: form.expires_at ? new Date(form.expires_at).toISOString() : undefined,
    })
    setShowAdd(false)
    setForm({ username: '', traffic_limit: 0, expires_at: '' })
    load()
  }

  const del = async (id: number) => {
    if (!confirm('Delete this user?')) return
    await api.delete(`/admin/users/${id}`)
    load()
  }

  const copySubLink = (token: string, id: number) => {
    navigator.clipboard.writeText(`${window.location.origin}/api/sub/${token}`)
    setCopied(id)
    setTimeout(() => setCopied(null), 2000)
  }

  const fmtTraffic = (bytes: number) => {
    if (bytes === 0) return 'Unlimited'
    const gb = bytes / (1024 * 1024 * 1024)
    return `${gb.toFixed(1)} GB`
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold">Users</h2>
        <button onClick={() => setShowAdd(!showAdd)} className="flex items-center gap-2 px-3 py-1.5 bg-white text-black rounded-lg text-sm font-medium hover:bg-zinc-200">
          <Plus size={14} /> Add User
        </button>
      </div>

      {showAdd && (
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 mb-4 grid grid-cols-3 gap-3">
          <input placeholder="Username" value={form.username} onChange={e => setForm({...form, username: e.target.value})} className="px-3 py-2 bg-black border border-zinc-700 rounded-lg text-sm text-white" />
          <input placeholder="Traffic Limit (GB, 0=unlimited)" type="number" value={form.traffic_limit} onChange={e => setForm({...form, traffic_limit: +e.target.value})} className="px-3 py-2 bg-black border border-zinc-700 rounded-lg text-sm text-white" />
          <input placeholder="Expires" type="date" value={form.expires_at} onChange={e => setForm({...form, expires_at: e.target.value})} className="px-3 py-2 bg-black border border-zinc-700 rounded-lg text-sm text-white" />
          <button onClick={add} className="px-3 py-2 bg-white text-black rounded-lg text-sm font-medium hover:bg-zinc-200 col-span-3">Save</button>
        </div>
      )}

      <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-zinc-800 text-zinc-500">
            <th className="text-left p-4">Username</th><th className="text-left p-4">Traffic</th><th className="text-left p-4">Expires</th><th className="text-left p-4">Status</th><th className="text-left p-4">Subscription</th><th className="p-4"></th>
          </tr></thead>
          <tbody>
            {users.map(u => (
              <tr key={u.ID} className="border-b border-zinc-800/50 hover:bg-zinc-800/30">
                <td className="p-4 font-medium">{u.username}</td>
                <td className="p-4 text-zinc-400">{fmtTraffic(u.traffic_used)} / {fmtTraffic(u.traffic_limit)}</td>
                <td className="p-4 text-zinc-400">{u.expires_at ? new Date(u.expires_at).toLocaleDateString() : 'Never'}</td>
                <td className="p-4"><span className={`px-2 py-0.5 rounded text-xs ${u.enabled ? 'bg-green-500/10 text-green-400' : 'bg-red-500/10 text-red-400'}`}>{u.enabled ? 'Active' : 'Disabled'}</span></td>
                <td className="p-4">
                  <button onClick={() => copySubLink(u.sub_token, u.ID)} className="flex items-center gap-1 text-zinc-400 hover:text-white text-xs">
                    {copied === u.ID ? <><Check size={12} /> Copied</> : <><Copy size={12} /> Copy Link</>}
                  </button>
                </td>
                <td className="p-4 text-right"><button onClick={() => del(u.ID)} className="text-zinc-500 hover:text-red-400"><Trash2 size={14} /></button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
```

- [ ] **Step 9: Build frontend**

```bash
cd web
npm run build
```

- [ ] **Step 10: Commit**

```bash
cd ..
git add .
git commit -m "feat: React frontend with Vercel-style dark UI"
```

---

### Task 10: Docker + Embed Frontend

**Files:**
- Create: `Dockerfile`
- Create: `docker-compose.yaml`
- Modify: `main.go` (serve frontend)

- [ ] **Step 1: Create Dockerfile**

```dockerfile
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.21-alpine AS backend
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=1 go build -o hy2board .

FROM alpine:latest
RUN apk add --no-cache ca-certificates sqlite-libs
WORKDIR /app
COPY --from=backend /app/hy2board .
COPY config.yaml .
EXPOSE 8080
CMD ["./hy2board"]
```

- [ ] **Step 2: Create docker-compose.yaml**

```yaml
version: "3"
services:
  hy2board:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./config.yaml:/app/config.yaml
    restart: unless-stopped
```

- [ ] **Step 3: Update main.go to serve frontend**

Uncomment the static file lines in `main.go` and update:

```go
	// Serve frontend
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/favicon.ico", "./web/dist/favicon.ico")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})
```

- [ ] **Step 4: Commit**

```bash
git add .
git commit -m "feat: Docker deployment + embedded frontend"
```

---

## Deliverable

Phase 1 完成后你会得到：

- **管理后台**：`http://your-server:8080` — 黑色 Vercel 风格，管理节点和用户
- **订阅链接**：`http://your-server:8080/api/sub/{token}` — 自动识别客户端返回 Surge/Clash/URI 格式
- **节点健康检测**：后台每 30 秒自动探测，不健康的节点不会出现在订阅中
- **Docker 一键部署**：`docker-compose up -d`
