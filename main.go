package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Feedback struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	Contact string `json:"contact" gorm:"type:varchar(255);not null"`
	Content string `json:"content" gorm:"type:text;not null"`
}

var db *gorm.DB

func mustGetEnv(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Fatalf("Environment variable %s is required but not set", key)
	return ""
}

func main() {
	dbHost := mustGetEnv("DB_HOST")
	dbPort := mustGetEnv("DB_PORT")
	dbUser := mustGetEnv("DB_USER")
	dbPass := mustGetEnv("DB_PASSWORD")
	dbName := mustGetEnv("DB_NAME")

	port := getEnv("PORT", "8080")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		dbHost, dbUser, dbPass, dbName, dbPort)

	var err error

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err := db.AutoMigrate(&Feedback{}); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}
	log.Println("Database connected and migrated")

	r := gin.Default()

	r.GET("/ping", healthCheck)

	r.POST("/saveFeedback", addFeedback)
	r.GET("/getLastFeedback", getLastFeedback)
	r.GET("/getAllFeedback", getAllFeedback)

	r.Run(":" + port)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

type FeedbackInput struct {
	Contact string `json:"contact"`
	Content string `json:"content" binding:"required"`
}

func addFeedback(c *gin.Context) {
	var input FeedbackInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	feedback := Feedback{
		Contact: input.Contact,
		Content: input.Content,
	}

	if err := db.Create(&feedback).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feedback"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Feedback saved"})
}

func getLastFeedback(c *gin.Context) {
	var feedback Feedback
	err := db.Order("id desc").First(&feedback).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "No feedback found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"feedback": feedback})
}

func getAllFeedback(c *gin.Context) {
	var feedback []Feedback
	if err := db.Find(&feedback).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"feedback": feedback})
}

func healthCheck(c *gin.Context) {
	sqlDB, err := db.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Failed to get DB instance"})
		return
	}
	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Database unreachable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Service is healthy",
		"time":    time.Now().UTC(),
	})
}
