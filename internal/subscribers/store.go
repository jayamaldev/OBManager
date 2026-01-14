package subscribers

import (
	"log/slog"
	"slices"
	"sync"

	"github.com/gorilla/websocket"
)

type UserStore struct {
	usersList map[*websocket.Conn]*User
	mu        sync.Mutex
}

func NewUserStore() *UserStore {
	usersList := make(map[*websocket.Conn]*User)

	return &UserStore{
		usersList: usersList,
	}
}

func (s *UserStore) AddUser(conn *websocket.Conn, user *User) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.usersList[conn] = user
}

func (s *UserStore) RemoveUser(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.usersList, conn)
}

func (s *UserStore) SubUser(conn *websocket.Conn, currPair string, sub bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subUser := s.usersList[conn]
	if sub {
		subUser.currPairs = append(subUser.currPairs, currPair)
	} else {
		for id, curr := range subUser.currPairs {
			if curr == currPair {
				subUser.currPairs = slices.Delete(subUser.currPairs, id, id+1)
			}
		}
	}

	slog.Info("User Subscription", "List", subUser.currPairs)
}

// GetSubscribedUsedList get list of users subscribed to the currency pair.
func (s *UserStore) GetSubscribedUsedList(currency string) []*User {
	var subUsersList []*User

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, user := range s.usersList {
		if slices.Contains(user.currPairs, currency) {
			subUsersList = append(subUsersList, user)
		}
	}

	return subUsersList
}
