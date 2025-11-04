package models

// TeamStat represents a team's score and name (used across models)
type TeamStat struct {
	Name  string `json:"name" bson:"name"`
	Score int    `json:"score" bson:"score"`
}

type Team struct {
	ID   string `json:"id" bson:"_id"`
	Name string `json:"team_name" bson:"team_name"`
}
