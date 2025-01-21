package main

import (
  "github.com/gin-gonic/gin"
  "gorm.io/gorm"
  "gorm.io/driver/sqlite"
  "github.com/google/uuid"
  "strings"
  "fmt"
)

type Url struct {
  ID uint `gorm:"primarykey"`
  Link string `json:"link"`
  Shortened string `json:"shortened"`
}

func exists(db *gorm.DB, shortUrl string) bool {
  var url Url
  db.Table("links").Where("shortened = ?", shortUrl).First(&url)
  return url.ID != 0
}

func generateShortUrl(db *gorm.DB) string {
  url := strings.Split(uuid.New().String(), "-")[0]
  if exists(db, url) {
    return generateShortUrl(db)
  }
  return url
}

func UrlLink(urll string, db *gorm.DB) string {
  var query Url
  fmt.Println(urll)
  db.Table("links").Where("shortened = ?", urll).First(&query)
  fmt.Println(query)
  if query.ID == 0 {
    return ""
  }
  return query.Link
}

func main() {
  r := gin.Default()
  
  r.LoadHTMLGlob("templates/*.html")

  db, err := gorm.Open(sqlite.Open("m.db"), &gorm.Config{})
  if err != nil {
    panic("failed to connect database")
  }

  r.GET("/", func(c *gin.Context) {
    c.HTML(200, "index.html", nil)
  })

  r.GET("/api/v1/shorten/:url", func(c *gin.Context) {
    url := c.Param("url")
    if len(url) > 45 || len(url) < 3 {
      c.JSON(200, gin.H{
        "erorr": "Link too long or too short.",
      })
      return
    }

    if !strings.Contains(url, "."){
      c.JSON(200, gin.H{
        "error": "Link doesn't have an TLD.",
      })
      return
    }
    shortUrl := generateShortUrl(db)

    if !strings.Contains(url, "http://") || !strings.Contains(url, "https://") {
      url = "https://" + url
    }

    db.Table("links").Create(&Url{Link: url, Shortened: shortUrl})

    c.JSON(200, gin.H{
      "url": url,
      "shortenUrl": c.Request.Host + "/" + shortUrl,
      "baseShortUrl": shortUrl,
    })
  })

  r.GET("/:shortUrl", func(c *gin.Context) {
    shortUrl := c.Param("shortUrl")
    if len(shortUrl) != 8 { return } 
    query := UrlLink(shortUrl, db)
    if query == "" {
      c.Redirect(302, "/")
      return
    }
    c.Redirect(302, query)
  }) 
  
  r.POST("/:shortUrl", func(c *gin.Context) {
    shortUrl := c.Param("shortUrl")
    if len(shortUrl) != 8 { return } 
    query := UrlLink(shortUrl, db)
    if query == "" {
      c.Redirect(302, "/")
      return
    }
    c.Redirect(302, query)
  })


  r.Run(":8080")
}
