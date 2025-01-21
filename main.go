package main

import (
  "github.com/gin-gonic/gin"
  "gorm.io/gorm"
  "gorm.io/driver/sqlite"
  "strconv"
  "html"
  "crypto/sha256"
  "strings"
  "encoding/hex"
  "fmt"
  "github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/didip/tollbooth_gin"
)

type Post struct {
  Id int `gorm:"primaryKey"`
  Title string `json:"title"`
  Poster string `json:"poster"`
  Board string `json:"board"`
  Body string `json:"body"`
  CreatedAt string `json:"created_at"`
  UpdatedAt string `json:"updated_at"`
}

type Admins struct {
  Id int `gorm:"primaryKey"`
  Authtoken string `json:"authtoken"`
  Username string `json:"username"`
  Password string `json:"password"`
  Boards int `json:"boards"`
  Description string `json:"description"`
  CreatedAt string `json:"created_at"`
  UpdatedAt string `json:"updated_at"`
}


func main() {
  r := gin.Default()
  metadata := []string{"A4Bet's Wonderland", "1", "A4Bet"} // first is name, second is board number, third is username of board creator

  db, err := gorm.Open(sqlite.Open("m.db"), &gorm.Config{})
  if err != nil {
    panic(err)
  }

  r.Use(gin.Recovery())

  lmt := tollbooth.NewLimiter(5, nil)
	lmt = tollbooth.NewLimiter(5, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})

	r.Use(tollbooth_gin.LimitHandler(lmt))

  r.LoadHTMLGlob("templates/*.html")

  db.AutoMigrate(&Post{})

  r.POST("/api/v1/post", func(c *gin.Context) {
    post := c.PostForm("post")
    title := c.PostForm("title")
    board := c.PostForm("board")
    authToken, err := c.Cookie("authToken")
    if err != nil {
      c.JSON(401, gin.H{
      "error": "unauthorized",
      })
      return
    }
    var admin Admins
    db.Table("admins").Where("authtoken = ?", authToken).First(&admin)
    if admin.Username == "" {
      c.JSON(403, gin.H{
      "error": "forbidden",
      })
      return
    }
    var postB Post = Post{
      Title: title,
      Poster: admin.Username,
      Board: board,
      Body: post,
    }
    if board == "1" && ( admin.Boards == 1 || admin.Boards == 3 ) {
      db.Table("posts").Create(&postB)
      c.JSON(200, gin.H{
        "message": "success",
      })
    } else if board == "2" && (admin.Boards == 2 || admin.Boards == 3) {
      db.Table("posts").Create(&postB)
      c.JSON(200, gin.H{
        "message": "success",
      })
    } else {
      c.JSON(403, gin.H{
      "error": "forbidden",
      })
    }

  })

  r.POST("/api/v1/getposts/:board/:limit", func(c *gin.Context) {
    var posts []Post
    boardID := c.Param("board")
    limitNum,err := strconv.Atoi(c.Param("limit"))
    if err != nil {
      c.JSON(400, gin.H{
      "error": "bad request",
      })
      return
    }
    if boardID != "1" && boardID != "2" && boardID != "3" {
      c.JSON(400, gin.H{
      "error": "bad request",
      })
      return
    }
    var rowcount int64
    if boardID == "3" {
      result := db.Model(&Post{}).Count(&rowcount)
      if result.Error != nil {
        c.JSON(400, gin.H{
        "error": "bad request",
        })
        return
      }
      db.Table("posts").Order("id desc").Limit(limitNum).
Find(&posts)
      for i := 0; i < len(posts); i++ {
      posts[i].Body = html.EscapeString(posts[i].Body)
      posts[i].Title = html.EscapeString(posts[i].Title)

    }
      c.JSON(200, gin.H{
        "posts": posts,
       "count": rowcount,
        "postsLen": len(posts),
      })
      return
    }
    result := db.Model(&Post{}).Where("board = ?", boardID).Count(&rowcount)
    if result.Error != nil {
      c.JSON(400, gin.H{
      "error": "bad request",
      })
      return
    }
    db.Table("posts").Where("board = ?", boardID).Order("id desc").Limit(limitNum).Find(&posts)
    for i := 0; i < len(posts); i++ {
      posts[i].Body = html.EscapeString(posts[i].Body)
      posts[i].Title = html.EscapeString(posts[i].Title)

    }
    c.JSON(200, gin.H{
      "posts": posts,
      "count": rowcount,
      "postsLen": len(posts),
    })
  })
  var adminofboard Admins
  if metadata[1] == "1" {
    db.Table("admins").Where("username = ?", "A4Bet").First(&adminofboard)
  }else if metadata[1] == "2" {
    db.Table("admins").Where("username = ?", "Kornballer").First(&adminofboard)
  }
  r.GET("/", func(c *gin.Context) {
    c.HTML(200, "index.html", gin.H{
      "title": metadata[0],
      "description": adminofboard.Description,
      "board": metadata[1],
    })
  })

  r.GET("/auth/admin", func(c *gin.Context) {
    authToken,err := c.Cookie("authToken")
    if err == nil && authToken != "notset" {
      var admin Admins
      db.Table("admins").Where("authtoken = ?", authToken).First(&admin)
      if admin.Username == "" {
        c.HTML(200, "admin.html", gin.H{
          "needlogin": true,
        })
        return
      }else{
        c.HTML(200, "admin.html", gin.H{
          "needlogin": false,
          "boardasigned": admin.Boards,
        })
      }
    }else{
      c.SetCookie("authToken", "notset", 3600, "/", "localhost", false, true)
      c.HTML(200, "admin.html", gin.H{
        "needlogin": true,
      })
    }
  })

  r.POST("/api/v1/authenticate/login", func(c *gin.Context) {
    username := c.PostForm("username")
    password := c.PostForm("password")
    var admin Admins
    result := db.Table("admins").Where("username = ?", username).First(&admin)
    if result.Error != nil {
      c.JSON(401, gin.H{
      "error": "unauthorized",
      })
      return
    }
    if (admin.Username == "" || username == "" || password == "" ){
      c.JSON(401, gin.H{
      "error": "unauthorized",
      })
      return
    }

    sha256Password := sha256.New()
    saltpss := strings.Split(admin.Password, "$")[0]
    sha256Password.Write([]byte(saltpss + password))
    HashedBytes := sha256Password.Sum(nil)
    Hashedpassword := hex.EncodeToString(HashedBytes)
    fmt.Println(Hashedpassword)
    fmt.Println(strings.Split(admin.Password, "$")[1])
    if strings.Split(admin.Password, "$")[1] != Hashedpassword {
      c.JSON(401, gin.H{
      "error": "unauthorized",
      })
      return
    }
    fmt.Println(admin.Authtoken)
    c.SetCookie("authToken", admin.Authtoken, 9600, "/", "localhost", false, true)
    c.JSON(200, gin.H{
    })
  })

  r.POST("/api/v1/description", func(c *gin.Context) {
    authToken, err := c.Cookie("authToken")
    if err != nil {
      c.JSON(401, gin.H{
      "error": "unauthorized",
      })
      return
    }
    var admin Admins
    db.Table("admins").Where("authtoken = ?", authToken).First(&admin)
    if admin.Username == "" {
      c.JSON(403, gin.H{
      "error": "forbidden",
      })
      return
    }
    description := c.PostForm("description")
    db.Table("admins").Where("authtoken = ?", authToken).Update("description", description)
    c.JSON(200, gin.H{
      "message": "success",
    })
  })

  r.GET("/api/v1/getdescription", func(c *gin.Context) {
    if metadata[1] == "1" {
      var adminofboard Admins
      db.Table("admins").Where("username = ?", "A4Bet").First(&adminofboard)
      c.JSON(200, gin.H{
        "description": adminofboard.Description,
      })
      return
    } else if metadata[1] == "2" {
      var adminofboard Admins
      db.Table("admins").Where("username = ?", "Kornballer").First(&adminofboard)
      c.JSON(200, gin.H{
        "description": adminofboard.Description,
      })
      return
    }
  })



  r.Run(":8080")
}
