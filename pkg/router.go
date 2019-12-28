package pkg

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	ws "github.com/gorilla/websocket"

	"github.com/gin-gonic/gin"
)

var (
	games = make(map[string]Game)

	upgrader = ws.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

type Game struct {
	ID        string
	PlayerIDs []string
	Conns     []ws.Conn
	Prompt    *Prompt
	Votes     map[string]int
}

type Prompt struct {
	Text      string
	Responses map[string]string
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

func NewRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.POST("/games", createGame)
	r.POST("/games/:id", joinGame)
	r.Any("/ws/games/:id", subscribe)
	r.POST("/games/:id/start", startGame)
	r.POST("/games/:id/response", submitResponse)
	r.POST("/games/:id/vote", vote)

	return r
}

func createGame(ctx *gin.Context) {
	id := generateGameId(4)
	games[id] = Game{id, make([]string, 0), make([]ws.Conn, 0), nil, nil}
	ctx.Header("Location", "/games/"+id)
}

type JoinGameReq struct {
	PlayerID string `json:"player_id"`
}

func joinGame(ctx *gin.Context) {
	var req JoinGameReq
	err := ctx.BindJSON(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PlayerID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "player_id must be non-empty"})
		return
	}

	id := ctx.Param("id")
	if id == "" {
		panic("id should never be null")
	}

	g, ok := games[id]
	if !ok {
		errMsg := fmt.Sprintf("game id %s does not exist", id)
		ctx.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		return
	}

	g.PlayerIDs = append(g.PlayerIDs, req.PlayerID)
	games[id] = g

	leader := true
	if len(g.PlayerIDs) > 1 {
		leader = false
	}
	ctx.JSON(http.StatusOK, gin.H{"leader": leader})
}

func subscribe(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		panic("id should never be null")
	}

	g, ok := games[id]
	if !ok {
		errMsg := fmt.Sprintf("game id %s does not exist", id)
		ctx.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		return
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	g.Conns = append(g.Conns, *conn)
	games[id] = g
}

type PromptMsg struct {
	Type   string `json:"type"`
	Prompt string `json:"prompt"`
}

func startGame(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		panic("id should never be null")
	}

	g, ok := games[id]
	if !ok {
		errMsg := fmt.Sprintf("game id %s does not exist", id)
		ctx.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		return
	}

	p, err := generatePrompt()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	g.Prompt = p
	games[id] = g

	msg := &PromptMsg{Type: "prompt", Prompt: p.Text}
	b, err := json.Marshal(msg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	g.Publish(b)
}

type SubmitResponseReq struct {
	PlayerID string `json:"player_id"`
	Response string `json:"response"`
}

type ResponsesSubmittedMsg struct {
	Type      string            `json:"type"`
	Responses map[string]string `json:"responses"`
}

func submitResponse(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		panic("id should never be null")
	}

	g, ok := games[id]
	if !ok {
		errMsg := fmt.Sprintf("game id %s does not exist", id)
		ctx.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		return
	}

	var req SubmitResponseReq
	err := ctx.BindJSON(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	g.Prompt.Responses[req.PlayerID] = req.Response
	games[id] = g

	if len(g.Prompt.Responses[req.PlayerID]) == len(g.PlayerIDs) {
		msg := &ResponsesSubmittedMsg{Type: "responses_submitted", Responses: g.Prompt.Responses}
		b, err := json.Marshal(msg)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		g.Votes = make(map[string]int)
		for _, pid := range g.PlayerIDs {
			g.Votes[pid] = 0
		}
		g.Publish(b)
	}
}

type SubmitVoteReq struct {
	Vote string `json:"vote"`
}

type VotesSubmittedMsg struct {
	Type  string         `json:"type"`
	Votes map[string]int `json:"votes"`
}

func vote(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		panic("id should never be null")
	}

	g, ok := games[id]
	if !ok {
		errMsg := fmt.Sprintf("game id %s does not exist", id)
		ctx.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		return
	}

	var req SubmitVoteReq
	err := ctx.BindJSON(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	g.Votes[req.Vote] += 1

	sum := 0
	for _, v := range g.Votes {
		sum += v
	}
	if sum == len(g.PlayerIDs) {
		msg := &VotesSubmittedMsg{Type: "votes_submitted", Votes: g.Votes}
		b, err := json.Marshal(msg)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		g.Publish(b)
	}
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

func generatePrompt() (*Prompt, error) {
	return &Prompt{
		Text:      "Welcome to Saint John!",
		Responses: make(map[string]string),
	}, nil
}
