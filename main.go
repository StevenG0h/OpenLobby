package main

import (
	"OpenLobby/queue"
	"OpenLobby/utils"
	"log"

	"github.com/gofiber/contrib/monitor"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/google/uuid"
)

func main() {
	app := fiber.New()

	env, err := utils.LoadConfig()

	if err != nil {
		log.Fatal(err)
		log.Fatal("Can't start OpenLobby due to invalid env config")
		return
	}

	waitingRoom := queue.NewWaitingRoom(env.NumberOfAllowedUsers)

	app.Use(session.New())

	app.Use(cors.New(cors.Config{
		AllowOrigins:     env.AllowedOrigins,
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

	go waitingRoom.RemoveExpiredSession(env.RemoveExpiredSession)

	log.Fatal(app.Listen(":" + env.Port))
}

func requestToken(wr *queue.WaitingRoom, c fiber.Ctx) error {
	session := session.FromContext(c)

	sessionId, ok := session.Get("userId").(string)
	if !ok {
		userId := uuid.New()

		token, isPassThrough, err := wr.PassthroughJoin(userId.String())

		if isPassThrough {
			session.Set("userId", userId.String())
			return c.JSON(fiber.Map{
				"token":   token,
				"session": userId.String(),
			})
		}

		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		wr.Join(userId.String())
		session.Set("userId", userId.String())

		return c.JSON(fiber.Map{
			"message": "User has joined the queue please wait",
		})
	}
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
	session := session.FromContext(c)

	sessionId, ok := session.Get("userId").(string)

	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "No session please redirect back")
	}

	if token == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Token is required")
	}

	isValid := wr.IsTokenValid(sessionId, token)
	if !isValid {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid or expired token")
	}

	return c.JSON(fiber.Map{
		"message": "Token is valid",
	})
}

func invalidateToken(wr *queue.WaitingRoom, c fiber.Ctx) error {
	token := c.Query("token")
	session := session.FromContext(c)

	sessionId, ok := session.Get("userId").(string)

	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "No session please redirect back")
	}

	if token == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Token is required")
	}

	err := wr.InvalidateToken(sessionId, token)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid or expired token")
	}

	return c.JSON(fiber.Map{
		"message": "Token invalidated successfully",
	})
}
