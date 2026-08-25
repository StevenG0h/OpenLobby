package main

import (
	"OpenLobby/queue"
	"log"

	"github.com/gofiber/contrib/monitor"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/google/uuid"
)

func main() {
	app := fiber.New()

	waitingRoom := queue.NewWaitingRoom()

	app.Use(session.New())

	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5174"},
		AllowCredentials: true,
	}))

	app.Get("/metrics", monitor.New())

	app.Get("/", func(ctx fiber.Ctx) error {
		return ctx.SendFile("./views/index.html")
	})

	app.Get("/request", func(ctx fiber.Ctx) error {
		return requestToken(waitingRoom, ctx)
	})

	app.Get("/validate-token", func(ctx fiber.Ctx) error {
		return validateToken(waitingRoom, ctx)
	})

	app.Post("/invalidate-token", func(ctx fiber.Ctx) error {
		return invalidateToken(waitingRoom, ctx)
	})

	go waitingRoom.RemoveExpiredSession()

	log.Fatal(app.Listen(":3000"))
}

func requestToken(wr *queue.WaitingRoom, c fiber.Ctx) error {
	session := session.FromContext(c)

	if session.Get("userId") == "" {
		userId := uuid.New()
		wr.Join(userId.String())
		session.Set("userID", userId)
		return c.JSON(fiber.Map{
			"message": "User has joined the queue please wait",
		})
	}

	sessionId := session.Get("userId").(string)

	token, err := wr.GetUserToken(sessionId)

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"token": token,
	})
}

func validateToken(wr *queue.WaitingRoom, c fiber.Ctx) error {
	token := c.Query("token")
	session := c.Query("userId")
	if token == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Token is required")
	}

	isValid := wr.IsTokenValid(session, token)
	if !isValid {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid or expired token")
	}

	return c.JSON(fiber.Map{
		"message": "Token is valid",
	})
}

func invalidateToken(wr *queue.WaitingRoom, c fiber.Ctx) error {
	token := c.Query("token")
	session := c.Query("userId")
	if token == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Token is required")
	}

	err := wr.InvalidateToken(session, token)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid or expired token")
	}

	return c.JSON(fiber.Map{
		"message": "Token invalidated successfully",
	})
}
