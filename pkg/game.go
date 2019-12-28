package pkg

import (
	_ "encoding/json"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
)

var (
	games   = MemGameStore{make(map[string]Game)}
	prompts = MemPromptStore{make(map[string]Prompt)}
)

type GameStore interface {
	New() (*Game, error)
	Get(id string) (*Game, error)
	Update(id string, g *Game) error
}

type Game struct {
	ID        string    `json:"id"`
	PlayerIDs []string  `json:"player_ids"`
	Conns     []ws.Conn `json:"-"`
	PromptIDs []string  `json:"prompt_ids"`
}

type MemGameStore struct {
	store map[string]Game
}

func (s *MemGameStore) New() (*Game, error) {
	id := generateGameId(4)
	g := &Game{
		id,
		make([]string, 0),
		make([]ws.Conn, 0),
		make([]string, 0),
	}

	s.store[id] = *g
	return g, nil
}

func (s *MemGameStore) Get(id string) (*Game, error) {
	g, ok := s.store[id]
	if !ok {
		return nil, nil
	}
	return &g, nil
}

func (s *MemGameStore) Update(id string, g *Game) error {
	s.store[id] = *g
	return nil
}

func generateGameId(n int) string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("abcdefghijklmnopqrstuvwxyz" +
		"0123456789")
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String() // E.g. "ExcbsVQs"
}

func (g *Game) Publish(msg []byte) {
	for _, c := range g.Conns {
		go publish(c, msg)
	}
}

func publish(conn ws.Conn, msg []byte) {
	w, err := conn.NextWriter(ws.TextMessage)
	if err != nil {
		log.Println(err)
		return
	}
	w.Write(msg)
	if err = w.Close(); err != nil {
		log.Println(err)
	}
}

type Prompt struct {
	ID        string
	Text      string
	Responses map[string]string
	Votes     map[string]int
}

type PromptStore interface {
	New() (*Prompt, error)
	Get(id string) (*Prompt, error)
	Update(id string, p *Prompt) error
}

type MemPromptStore struct {
	store map[string]Prompt
}

func (s *MemPromptStore) New() (*Prompt, error) {
	p := Prompt{
		ID:        uuid.New().String(),
		Text:      "Welcome to Saint John!",
		Responses: make(map[string]string),
		Votes:     make(map[string]int),
	}
	s.store[p.ID] = p
	return &p, nil
}

func (s *MemPromptStore) Get(id string) (*Prompt, error) {
	g, ok := s.store[id]
	if !ok {
		return nil, nil
	}
	return &g, nil
}

func (s *MemPromptStore) Update(id string, g *Prompt) error {
	s.store[id] = *g
	return nil
}
