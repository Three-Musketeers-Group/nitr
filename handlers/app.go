package handlers

import (
	"encoding/json"
	"log"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/bitcav/nitr-core/host"
	db "github.com/bitcav/nitr/database"
	"github.com/bitcav/nitr/models"
	"github.com/bitcav/nitr/utils"
	"github.com/gofiber/fiber"
	"github.com/gofiber/session"
	"github.com/gofiber/websocket"
	"github.com/hoisie/mustache"
)

var sessions = session.New()

func Login(c *fiber.Ctx) {
	store := sessions.Get(c)
	if store.Get("UserID") == "1" || c.Cookies("remember") == "1" {
		c.Redirect("/panel")
	} else {
		content, err := rice.MustFindBox("../app/views").HTTPBox().String("login.mustache")
		utils.LogError(err)

		layout, err := rice.MustFindBox("../app/views/layout").HTTPBox().String("default.mustache")
		utils.LogError(err)

		c.Type("html")
		c.Send(mustache.RenderInLayout(content, layout))
	}
}

func LoginSubmit(c *fiber.Ctx) {
	login := new(models.Login)

	if err := c.BodyParser(login); err != nil {
		log.Fatal(err)
	}

	nitrUser := db.GetUserByID("1")
	if (login.Username == nitrUser.Username) && (login.Password == nitrUser.Password) {
		store := sessions.Get(c)
		defer store.Save()
		store.Set("UserID", "1")
		if login.Remember == "on" {
			cookie := new(fiber.Cookie)
			cookie.Name = "remember"
			cookie.Value = "1"
			cookie.Expires = time.Now().Add(48 * time.Hour)
			c.Cookie(cookie)
		}
		c.Redirect("/panel")
	} else {
		c.Redirect("/")
	}
}

func Panel(c *fiber.Ctx) {
	content, err := rice.MustFindBox("../app/views").HTTPBox().String("panel.html")
	utils.LogError(err)
	layout, err := rice.MustFindBox("../app/views/layout").HTTPBox().String("default.mustache")
	utils.LogError(err)

	c.Type("html")
	c.Send(mustache.RenderInLayout(content, layout))

	log.Println("Session started")
}

func PanelContent(c *fiber.Ctx) {
	hostInfo := models.HostInfo{
		Name:        host.Info().Name,
		Description: host.Info().Platform + "/" + host.Info().Arch,
		IP:          utils.GetLocalIP(),
		Port:        utils.GetLocalPort(),
		Key:         db.GetApiKey(),
	}

	hostInfoJSON, err := json.Marshal(hostInfo)
	if err != nil {
		utils.LogError(err)
	}

	hostInfo.QrCode = string(hostInfoJSON)

	c.JSON(hostInfo)
}

func Logout(c *fiber.Ctx) {
	c.ClearCookie()
	c.Redirect("/")
	log.Println("Session closed")
}

func GenerateApiKey(c *fiber.Ctx) {
	newAPIKey := utils.RandString(10)

	hostInfo := models.HostInfo{
		Name:        host.Info().Name,
		Description: host.Info().Platform + "/" + host.Info().Arch,
		IP:          utils.GetLocalIP(),
		Port:        utils.GetLocalPort(),
		Key:         newAPIKey,
	}

	hostInfoJSON, err := json.Marshal(hostInfo)
	if err != nil {
		utils.LogError(err)
	}

	nitrUser := db.GetUserByID("1")
	user := models.User{Username: nitrUser.Username, Password: nitrUser.Password, Apikey: newAPIKey}
	err = db.SetUserData("1", user)
	utils.LogError(err)

	c.JSON(models.ApiKey{
		Key:    newAPIKey,
		QrCode: string(hostInfoJSON),
	})

	log.Println("New Api key generated")
}

func Password(c *fiber.Ctx) {
	content, err := rice.MustFindBox("../app/views").HTTPBox().String("password.html")
	utils.LogError(err)
	layout, err := rice.MustFindBox("../app/views/layout").HTTPBox().String("default.mustache")
	utils.LogError(err)

	c.Type("html")
	c.Send(mustache.RenderInLayout(content, layout))
}

func PasswordSubmit(c *fiber.Ctx) {
	password := new(models.Password)

	if err := c.BodyParser(password); err != nil {
		log.Fatal(err)
	}

	nitrUser := db.GetUserByID("1")

	if password.CurrentPassword == nitrUser.Password {
		user := models.User{Username: nitrUser.Username, Password: password.NewPassword, Apikey: nitrUser.Apikey}
		err := db.SetUserData("1", user)
		utils.LogError(err)
		c.SendStatus(200)
		log.Println("Password changed")
	} else {
		c.SendStatus(304)
	}
}

func SocketReader(c *websocket.Conn) {
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		log.Printf("%s", msg)
	}

}

//Auth Middleware
func Auth(c *fiber.Ctx) {
	store := sessions.Get(c)
	if store.Get("UserID") == "1" || c.Cookies("remember") == "1" {
		c.Next()
	} else {
		c.Redirect("/")
	}
}