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
	Prompt string `json:"prompt"`
}

func (g *Game) PublishPrompt(text string) {
	msg := &PromptMsg{
		Type:   "prompt",
		Prompt: text,
	}
	g.Publish(msg)
}

type ResponsesSubmittedMsg struct {
	Type      string            `json:"type"`
	Responses map[string]string `json:"responses"`
}

func (g *Game) PublishResponses(responses map[string]string) {
	msg := &ResponsesSubmittedMsg{
		Type:      "responses_submitted",
		Responses: responses,
	}
	g.Publish(msg)
}

type VotesSubmittedMsg struct {
	Type  string         `json:"type"`
	Votes map[string]int `json:"votes"`
}

func (g *Game) PublishVotes(votes map[string]int) {
	msg := &VotesSubmittedMsg{
		Type:  "votes_submitted",
		Votes: votes,
	}
	g.Publish(msg)
}
