package models

// Team structure
type Teamm struct {
	Name  string `json:"name" bson:"name"`
	Score int    `json:"score" bson:"score"`
}


type TeamStats struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type Team struct {
	ID   string `json:"id" bson:"_id"`
	Name string `json:"team_name" bson:"team_name"`
}
