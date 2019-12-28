package pkg

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"
)

var (
	upgrader = ws.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

func NewRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.POST("/games", createGame)
	r.POST("/games/:gameId", joinGame)
	r.GET("/games/:gameId", getGame)
	r.Any("/ws/games/:gameId", subscribe)
	r.POST("/games/:gameId/start", startGame)
	r.POST("/games/:gameId/prompts/:promptId/response", submitResponse)
	r.POST("/games/:gameId/prompts/:promptId/vote", vote)

	return r
}

func createGame(ctx *gin.Context) {
	g, err := games.New()
	if err != nil {
		panic(err)
	}
	ctx.Header("Location", "/games/"+g.ID)
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

	id := ctx.Param("gameId")
	if id == "" {
		panic("id should never be null")
	}

	g, err := games.Get(id)
	if err != nil {
		panic(err)
	} else if g == nil {
		errMsg := fmt.Sprintf("game id %s does not exist", id)
		ctx.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		return
	}

	g.PlayerIDs = append(g.PlayerIDs, req.PlayerID)
	if err := games.Update(id, g); err != nil {
		panic(err)
	}

	leader := true
	if len(g.PlayerIDs) > 1 {
		leader = false
	}
	ctx.JSON(http.StatusOK, gin.H{"leader": leader})
}

func getGame(ctx *gin.Context) {
	id := ctx.Param("gameId")
	if id == "" {
		panic("id should never be null")
	}

	g, err := games.Get(id)
	if err != nil {
		panic(err)
	} else if g == nil {
		errMsg := fmt.Sprintf("game id %s does not exist", id)
		ctx.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		return
	}

	ctx.JSON(http.StatusOK, g)
}

func subscribe(ctx *gin.Context) {
	id := ctx.Param("gameId")
	if id == "" {
		panic("id should never be null")
	}

	g, err := games.Get(id)
	if err != nil {
		panic(err)
	} else if g == nil {
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
	if err := games.Update(id, g); err != nil {
		panic(err)
	}
}

type PromptMsg struct {
	Type   string `json:"type"`
	Prompt string `json:"prompt"`
}

func startGame(ctx *gin.Context) {
	id := ctx.Param("gameId")
	if id == "" {
		panic("id should never be null")
	}

	g, err := games.Get(id)
	if err != nil {
		panic(err)
	} else if g == nil {
		errMsg := fmt.Sprintf("game id %s does not exist", id)
		ctx.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		return
	}

	p, err := prompts.New()
	if err != nil {
		panic(err)
	}
	g.PlayerIDs = []string{p.ID}
	games.Update(id, g)

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
	gameId := ctx.Param("gameId")
	if gameId == "" {
		panic("game id should never be null")
	}

	g, err := games.Get(gameId)
	if err != nil {
		panic(err)
	} else if g == nil {
		errMsg := fmt.Sprintf("game id %s does not exist", gameId)
		ctx.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		return
	}

	promptId := ctx.Param("promptId")
	if promptId == "" {
		panic("prompt id should never be null")
	}

	p, err := prompts.Get(promptId)
	if err != nil {
		panic(err)
	} else if p == nil {
		errMsg := fmt.Sprintf("prompt id %s does not exist", promptId)
		ctx.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		return
	}

	var req SubmitResponseReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p.Responses[req.PlayerID] = req.Response
	if err := prompts.Update(promptId, p); err != nil {
		panic(err)
	}

	if len(p.Responses) == len(g.PlayerIDs) {
		msg := &ResponsesSubmittedMsg{Type: "responses_submitted", Responses: p.Responses}
		b, err := json.Marshal(msg)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
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
	gameId := ctx.Param("gameId")
	if gameId == "" {
		panic("game id should never be null")
	}

	g, err := games.Get(gameId)
	if err != nil {
		panic(err)
	} else if g == nil {
		errMsg := fmt.Sprintf("game id %s does not exist", gameId)
		ctx.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		return
	}

	promptId := ctx.Param("promptId")
	if promptId == "" {
		panic("prompt id should never be null")
	}

	p, err := prompts.Get(promptId)
	if err != nil {
		panic(err)
	} else if p == nil {
		errMsg := fmt.Sprintf("prompt id %s does not exist", promptId)
		ctx.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		return
	}

	var req SubmitVoteReq
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p.Votes[req.Vote] += 1
	if err := prompts.Update(promptId, p); err != nil {
		panic(err)
	}

	sum := 0
	for _, v := range p.Votes {
		sum += v
	}
	if sum == len(g.PlayerIDs) {
		msg := &VotesSubmittedMsg{
			Type:  "votes_submitted",
			Votes: p.Votes,
		}

		b, err := json.Marshal(msg)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		g.Publish(b)
	}
}
