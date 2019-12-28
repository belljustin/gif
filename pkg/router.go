package pkg

import (
	_ "encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
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
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.ExposeHeaders = []string{"Content-Length", "Location"}
	r.Use(cors.New(config))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.POST("/games", createGame)
	r.POST("/games/:gameId", joinGame)
	r.GET("/games/:gameId", getGame)
	r.GET("/ws/games/:gameId", subscribe)
	r.POST("/games/:gameId/start", startGame)
	r.POST("/games/:gameId/next", next)
	r.GET("/games/:gameId/prompts/:promptId", getPrompt)
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

	g.PublishJoined(req.PlayerID)

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
	g.PromptIDs = []string{p.ID}
	games.Update(id, g)

	g.PublishPrompt(p.ID, p.Text)
}

func getPrompt(ctx *gin.Context) {
	promptId := ctx.Param("promptId")
	if promptId == "" {
		panic("game id should never be null")
	}

	p, err := prompts.Get(promptId)
	if err != nil {
		panic(err)
	} else if p == nil {
		errMsg := fmt.Sprintf("prompt id %s does not exist", promptId)
		ctx.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		return
	}

	ctx.JSON(http.StatusOK, p)
}

type SubmitResponseReq struct {
	PlayerID string `json:"player_id"`
	Response string `json:"response"`
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
		g.PublishResponses(p.Responses)
	}
}

type SubmitVoteReq struct {
	Vote string `json:"vote"`
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
		g.PublishVotes(p.Votes)
	}
}

func next(ctx *gin.Context) {
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

	if len(g.PromptIDs)%5 == 0 {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "not implemented"})
		return
	}

	p, err := prompts.New()
	if err != nil {
		panic(err)
	}
	g.PromptIDs = append(g.PromptIDs, p.ID)
	games.Update(gameId, g)

	g.PublishPrompt(p.ID, p.Text)
}
