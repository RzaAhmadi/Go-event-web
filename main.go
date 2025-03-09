package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/postgres"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var db *gorm.DB

type User struct {
	ID       uint   `gorm:"primary_key"`
	Username string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
}

type Event struct {
	ID             uint   `gorm:"primary_key"`
	Description    string `gorm:"not null"`
	Details        string `gorm:"not null"`
	Title          string `gorm:"not null"`
	RCANumber      string `gorm:"not null"`
	GroupName      string `gorm:"not null"`
	EventDate      string `gorm:"not null"`
	StartTime      string `gorm:"not null"`
	EndTime        string `gorm:"not null"`
	RegisteredUser string `gorm:"null"`
}

func main() {
	var err error
	// اتصال به پایگاه داده
	db, err = gorm.Open("postgres", "host=localhost user=postgres dbname=DB_Form_Web password=1 sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to the database: ", err)
	}
	defer db.Close()

	// خودکارسازی جداول
	db.AutoMigrate(&User{}, &Event{})

	r := gin.Default()

	// تنظیمات session
	store, err := postgres.NewStore(db.DB(), []byte("secret")) // ایجاد یک استور جدید
	if err != nil {
		log.Fatal("Failed to create session store: ", err)
	}
	r.Use(sessions.Sessions("mysession", store))

	r.LoadHTMLGlob("templates/*")

	// صفحات
	r.GET("/login", loginPage)
	r.POST("/login", login)
	r.POST("/add-user", AddUser)
	r.GET("/dashboard", dashboardPage)
	r.POST("/add-event", addEvent)
	r.GET("/events", eventsPage)
	r.POST("/edit-event/:id", editEvent)
	r.POST("/delete-event/:id", deleteEvent)
	r.POST("/edit-event-ajax/:id", editEventAjax)

	r.Run(":8080")
}

// سایر توابع loginPage، login، dashboardPage، addEvent و غیره...

func loginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

func login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	var user User
	if err := db.Where("username = ? AND password = ?", username, password).First(&user).Error; err == nil {
		session := sessions.Default(c)
		session.Set("user_id", user.ID) // ذخیره شناسه کاربر در جلسه
		session.Save()                  // ذخیره تغییرات در جلسه
		c.Redirect(http.StatusFound, "/dashboard")
	} else {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "نام کاربری یا رمز عبور نادرست است!"})
	}
}

func dashboardPage(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("user_id") == nil { // بررسی ورود کاربر
		c.Redirect(http.StatusFound, "/login")
		return
	}
	// دریافت اطلاعات کاربر از دیتابیس
	var user User
	if err := db.First(&user, user).Error; err != nil {
		// اگر کاربر پیدا نشد، به صفحه لاگین برگردونید یا یک پیام خطا نمایش بدید
		c.Redirect(http.StatusFound, "/login")
		return
	}

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"username": user.Username, // پاس دادن نام کاربری به قالب
	})
}

func addEvent(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("user_id") == nil { // بررسی ورود کاربر
		c.Redirect(http.StatusFound, "/login")
		return
	}

	startTime, err := formatTime(c.PostForm("start_time"))
	if err != nil {
		fmt.Println("Error formatting start time:", err)
		c.String(http.StatusBadRequest, "Invalid start time format")
		return
	}

	endTime, err := formatTime(c.PostForm("end_time"))
	if err != nil {
		fmt.Println("Error formatting end time:", err)
		c.String(http.StatusBadRequest, "Invalid end time format")
		return
	}

	event := Event{
		Description:    c.PostForm("description"),
		Details:        c.PostForm("details"),
		Title:          c.PostForm("title"),
		RCANumber:      c.PostForm("rca_number"),
		GroupName:      c.PostForm("group_name"),
		EventDate:      c.PostForm("date"),
		StartTime:      startTime,
		EndTime:        endTime,
		RegisteredUser: c.PostForm("registered_user"),
	}
	db.Create(&event)
	c.Redirect(http.StatusFound, "/dashboard")
}

func AddUser(c *gin.Context) {
	adduser := User{
		Username: c.PostForm("username"),
		Password: c.PostForm("password"),
	}
	db.Create(&adduser)
	c.Redirect(http.StatusFound, "login")
}

func eventsPage(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("user_id") == nil { // بررسی ورود کاربر
		c.Redirect(http.StatusFound, "/login")
		return
	}
	var events []Event
	db.Find(&events)
	c.HTML(http.StatusOK, "events.html", gin.H{"events": events})
}

func editEvent(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("user_id") == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	id := c.Param("id")
	var event Event
	if err := db.First(&event, id).Error; err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	event.Description = c.PostForm("description")
	event.Details = c.PostForm("details")
	event.Title = c.PostForm("title")
	event.RCANumber = c.PostForm("rca_number")
	event.GroupName = c.PostForm("group_name")
	event.EventDate = c.PostForm("event_date")
	event.RegisteredUser = c.PostForm("registered_user")

	startTime, err := formatTime(c.PostForm("start_time"))
	if err != nil {
		fmt.Println("Error formatting start time:", err)
		c.String(http.StatusBadRequest, "Invalid start time format")
		return
	}
	event.StartTime = startTime

	endTime, err := formatTime(c.PostForm("end_time"))
	if err != nil {
		fmt.Println("Error formatting end time:", err)
		c.String(http.StatusBadRequest, "Invalid end time format")
		return
	}
	event.EndTime = endTime

	db.Save(&event)
	c.Redirect(http.StatusFound, "/events")
}

// -----------------
func editEventAjax(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("user_id") == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	id := c.Param("id")
	var event Event
	if err := db.First(&event, id).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Event not found"})
		return
	}

	// Update event fields from form data
	event.Description = c.PostForm("description")
	event.RCANumber = c.PostForm("rca_number")
	event.GroupName = c.PostForm("group_name")
	event.Details = c.PostForm("details")
	event.Title = c.PostForm("title")
	event.EventDate = c.PostForm("event_date")

	startTime, err := formatTime(c.PostForm("start_time"))
	if err != nil {
		fmt.Println("Error formatting start time:", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid start time format"})
		return
	}
	event.StartTime = startTime

	endTime, err := formatTime(c.PostForm("end_time"))
	if err != nil {
		fmt.Println("Error formatting end time:", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid end time format"})
		return
	}
	event.EndTime = endTime

	// Save the updated event to the database
	if err := db.Save(&event).Error; err != nil {
		fmt.Println("Error saving event:", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "خطا در بروز رسانی"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Event updated successfully"})
}

//-----------------

func deleteEvent(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("user_id") == nil { // بررسی ورود کاربر
		c.Redirect(http.StatusFound, "/login")
		return
	}
	id := c.Param("id")
	db.Delete(&Event{}, id)
	c.Redirect(http.StatusFound, "/events")
}

func formatTime(timeString string) (string, error) {
	// Assuming the timeString is in the format "HH:MM"
	t, err := time.Parse("15:04", timeString)
	if err != nil {
		return "", err
	}
	return t.Format("15:04:05"), nil // Return in "HH:MM:SS" format
}

// Handler for logout
func logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Options(sessions.Options{MaxAge: -1}) // تنظیم MaxAge به -1 برای حذف session
	err := session.Save()
	if err != nil {
		log.Println("Error saving session after logout:", err) // ثبت خطا
	}
	c.Redirect(http.StatusFound, "/login")
}
