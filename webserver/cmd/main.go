package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
	"teste.weberser.com/webserver/chat"
)

type NewChatForm struct {
	ChatName string `form:"chatName" binding:"required"`
}

func main() {

	app := gin.Default()
	app.LoadHTMLGlob("../template/**/*")

	secretKey := make([]byte, 32)
	_, err := rand.Read(secretKey)
	if err != nil {
		log.Printf("error generating secret key: %v\n", err)
	}
	secretString := strings.TrimRight(strings.ReplaceAll(base64.StdEncoding.EncodeToString(secretKey), "=", ""), "-_")

	store := cookie.NewStore([]byte(secretString))
	app.Use(sessions.Sessions("TempData", store))

	chats := chat.NewChartServerColletion()

	chatAdd := make(chan string)

	chatGroup := app.Group("/chat")

	chatGroup.GET("/", func(c *gin.Context) {
		session := sessions.Default(c)
		notification := session.Get("message")
		status := session.Get("status")
		if notification != nil || status != nil {
			session.Clear()
			session.Save()
		}

		c.HTML(http.StatusOK, "chat/index.html", gin.H{
			"chats":   chats.GetChatServersKeys(),
			"status":  status,
			"message": notification,
		})
	})

	chatGroup.POST("/createchat", func(c *gin.Context) {
		//validar se exite chat criado com o nome
		session := sessions.Default(c)

		var newChat NewChatForm
		if err := c.ShouldBind(&newChat); err != nil {
			//set message from session
			session.Set("status", "error")
			session.Set("message", "Error binding form")
			c.Redirect(http.StatusSeeOther, "/chat")
			return
		}

		conn := chats.GetChatServer(newChat.ChatName)
		if conn != nil {
			//set message from session
			session.Set("status", "error")
			session.Set("message", "Chat name already exists")
			c.Redirect(http.StatusSeeOther, "/chat")
			return
		}

		chats.AddChatServer(newChat.ChatName)
		session.Set("status", "success")
		session.Set("message", "Chat created")
		session.Save()
		log.Printf("created chat %s\n", newChat.ChatName)

		c.Redirect(http.StatusSeeOther, "/chat")

		chatAdd <- newChat.ChatName //Sinalize to add new chat server
	})

	chatGroup.GET("/connect/:name", func(c *gin.Context) {

		chatName := c.Param("name")
		conn := chats.GetChatServer(chatName)
		if conn == nil {
			c.Redirect(http.StatusSeeOther, "/chat")
			return
		}
		c.HTML(http.StatusOK, "chat/chat.html", gin.H{
			"name": chatName,
		})
	})

	chatGroup.GET("/ws/:name", func(c *gin.Context) {
		chatName := c.Param("name")
		conn := chats.GetChatServer(chatName)
		if conn == nil {
			c.Redirect(http.StatusSeeOther, "/chat")
			//Set message to show on /chat endpoint
			return
		}
		websocket.Handler(conn.HandleConnections).ServeHTTP(c.Writer, c.Request)
	})

	server := &http.Server{
		Addr:           ":8080",          // listen on 0.0.0.0:8080
		Handler:        app,              // Pass our router
		ReadTimeout:    10 * time.Second, // 10 seconds
		WriteTimeout:   10 * time.Second, // 10 seconds
		MaxHeaderBytes: 5 << 20,          // 5 MB
	}

	go func() {
		log.Printf("Starting server on port :8080")
		if err := server.ListenAndServe(); err != nil && err == http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	//strt handMessages for chats
	go func() {
		for chat := range chatAdd {
			log.Printf("adding chat %s\n", chat)
			conn := chats.GetChatServer(chat)
			go conn.HandleMessages()
		}
	}()

	//check if chat is empty
	go chats.MonitorChatServers()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	log.Println("Application is Shutingdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
}
