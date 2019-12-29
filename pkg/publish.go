package pkg

type JoinedMsg struct {
	Type     string `json:"type"`
	PlayerID string `json:"player_id"`
}

func (g *Game) PublishJoined(playerID string) {
	msg := JoinedMsg{
		Type:     "joined",
		PlayerID: playerID,
	}
	g.Publish(msg)
}

type PromptMsg struct {
	Type   string `json:"type"`
	ID     string `json:"id"`
	Prompt string `json:"prompt"`
}

func (g *Game) PublishPrompt(id, text string) {
	msg := &PromptMsg{
		Type:   "prompt",
		ID:     id,
		Prompt: text,
	}
	g.Publish(msg)
}

type ResponsesSubmittedMsg struct {
	Type      string            `json:"type"`
	Responses map[string]string `json:"responses"`
    ID        string            `json:"id"`
}

func (g *Game) PublishResponses(responses map[string]string, promptID string) {
	msg := &ResponsesSubmittedMsg{
		Type:      "responses_submitted",
		Responses: responses,
        ID:        promptID,
	}
	g.Publish(msg)
}

type VotesSubmittedMsg struct {
	Type  string         `json:"type"`
	Votes map[string]int `json:"votes"`
    ID    string         `json:"id"`
}

func (g *Game) PublishVotes(votes map[string]int, promptID string) {
	msg := &VotesSubmittedMsg{
		Type:  "votes_submitted",
		Votes: votes,
        ID: promptID,
	}
	g.Publish(msg)
}

type EndMsg struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func (g *Game) PublishEnd(ID string) {
	msg := &EndMsg{
		Type: "end",
		ID:   ID,
	}
	g.Publish(msg)
}
