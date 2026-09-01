package queue

import (
	"OpenLobby/utils"
	"errors"
	"sync"
	"time"
)

type User struct {
	ID        string
	token     string
	ExpiredAt time.Time
	JoinedAt  time.Time
}

type WaitingRoom struct {
	mu         sync.Mutex
	users      map[string]User
	order      []string
	activeUser map[string]User
}

func NewWaitingRoom() *WaitingRoom {
	return &WaitingRoom{
		users:      make(map[string]User),
		activeUser: make(map[string]User),
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

func (wr *WaitingRoom) PopNext() (string, bool, error) {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	if len(wr.order) == 0 {
		return "", false, errors.New("Queue is empty")
	}

	nextID := wr.order[0]
	user := wr.users[nextID]

	user.ExpiredAt = time.Now().Add(15 * time.Minute)
	token, err := utils.GenerateRandomString(32)
	user.token = token

	if err != nil {
		return "", false, errors.New("Failed to generate token")
	}

	wr.activeUser[nextID] = user

	wr.order = wr.order[1:]

	delete(wr.users, nextID)

	return nextID, true, nil
}

func (wr *WaitingRoom) IsUserInLine(userID string) bool {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	_, exists := wr.users[userID]
	return exists
}

func (wr *WaitingRoom) GetUserToken(userID string) (string, error) {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	user, exists := wr.activeUser[userID]

	if !exists {
		return "", errors.New("User not in active list yet")
	}

	return user.token, nil
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

	delete(wr.users, userID)

	return nil
}

func (wr *WaitingRoom) RemoveExpiredSession() {
	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()

	for t := range ticker.C {
		println("Session Cleaner Is Running", t.Format(time.RFC1123))
		now := time.Now()
		wr.mu.Lock()

		for _, user := range wr.activeUser {
			isExpired := user.ExpiredAt.Nanosecond() < now.Nanosecond()

			if isExpired {
				delete(wr.users, user.ID)
				wr.PopNext()
			}
		}
		wr.mu.Unlock()

		println("Session Cleaning Is Complete")
		println("Number of user in waiting:", len(wr.users))
	}
}

func (wr *WaitingRoom) PassthroughJoin(userId string) (string, bool, error) {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	if !(len(wr.activeUser) < 50 && len(wr.order) == 0) {
		return "", false, nil
	}

	user := User{
		ExpiredAt: time.Now().Add(15 * time.Minute),
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
