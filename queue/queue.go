package queue

import (
	"OpenLobby/utils"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type User struct {
	ID        string
	token     string
	ExpiredAt time.Time
	JoinedAt  time.Time
}

type WaitingRoom struct {
	mu            sync.Mutex
	users         map[string]User
	order         []string
	activeUser    map[string]User
	maxUser       int
	tokenDuration int
}

func NewWaitingRoom(maxUser int, tokenDuration int) *WaitingRoom {
	return &WaitingRoom{
		users:         make(map[string]User),
		activeUser:    make(map[string]User),
		maxUser:       maxUser,
		tokenDuration: tokenDuration,
	}
}

func (wr *WaitingRoom) Join(userID string) {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	if _, exists := wr.users[userID]; exists {
		return
	}

	wr.users[userID] = User{
		ID:       userID,
		JoinedAt: time.Now(),
	}
	wr.order = append(wr.order, userID)
}

func (wr *WaitingRoom) PopNext() (string, error) {
	if len(wr.order) == 0 {
		return "", errors.New("Queue is empty")
	}

	if len(wr.activeUser) == wr.maxUser {
		return "", errors.New("Queue still full please wait")
	}

	nextID := wr.order[0]
	user := wr.users[nextID]

	user.ExpiredAt = time.Now().Add(time.Duration(wr.tokenDuration) * time.Second)
	token, err := utils.GenerateRandomString(32)
	user.token = token

	if err != nil {
		return "", errors.New("Failed to generate token")
	}

	wr.activeUser[nextID] = user

	wr.order = wr.order[1:]

	delete(wr.users, nextID)

	return nextID, nil
}

func (wr *WaitingRoom) IsUserInLine(userID string) bool {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	_, exists := wr.users[userID]
	return exists
}

func (wr *WaitingRoom) GetUserToken(userID string) string {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	user := wr.activeUser[userID]

	return user.token
}

func (wr *WaitingRoom) IsUserActive(userID string) bool {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	_, exists := wr.activeUser[userID]
	return exists
}

func (wr *WaitingRoom) IsTokenValid(userID string, token string) bool {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	user, exists := wr.activeUser[userID]

	if !exists {
		return false
	}

	if user.token != token {
		return false
	}

	return true
}

func (wr *WaitingRoom) InvalidateToken(userID string, token string) error {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	user, exists := wr.activeUser[userID]

	if !exists {
		return errors.New("The user not exists in active user list")
	}

	if user.token != token {
		return errors.New("Invalid token")
	}

	delete(wr.activeUser, userID)

	return nil
}

func (wr *WaitingRoom) RemoveExpiredSession(interval int, log *logrus.Logger) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)

	defer ticker.Stop()

	for t := range ticker.C {
		log.Info("Session Cleaner Is Running", t.Format(time.RFC1123))
		now := time.Now()
		wr.mu.Lock()

		numberOfActiveUsers := len(wr.activeUser)

		if numberOfActiveUsers == 0 && len(wr.users) != 0 {
			for i := 0; i < (wr.maxUser - numberOfActiveUsers); i++ {
				_, err := wr.PopNext()

				if err != nil {
					log.Error(err)
				}
			}
		}

		for _, user := range wr.activeUser {
			isExpired := user.ExpiredAt.Before(now)

			if isExpired {
				delete(wr.activeUser, user.ID)
				_, err := wr.PopNext()

				if err != nil {
					log.Error(err)
				}
			}
		}
		wr.mu.Unlock()

		log.Info("Session Cleaning Is Complete")
		log.Info("Number of user in waiting:", len(wr.users))
		log.Info("Number of active users:", len(wr.activeUser))
	}
}

func (wr *WaitingRoom) PassthroughJoin(userId string) (string, bool, error) {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	if !(len(wr.activeUser) < wr.maxUser && len(wr.order) == 0) {
		return "", false, nil
	}

	user := User{
		ExpiredAt: time.Now().Add(time.Duration(wr.tokenDuration) * time.Second),
		ID:        userId,
	}

	token, err := utils.GenerateRandomString(32)
	user.token = token

	if err != nil {
		return "", false, errors.New("Failed to generate token")
	}

	wr.activeUser[userId] = user

	return token, true, nil
}
