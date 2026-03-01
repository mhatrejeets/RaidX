package handlers

import "github.com/mhatrejeets/RaidX/internal/redisImpl"

func forceScorerTakeover(matchID, redirectURL string) {
	if matchID == "" {
		return
	}
	_ = redisImpl.DeleteRedisKey("scorer_lock:" + matchID)
	room := GetRoom(matchID)
	room.NotifyAndCloseScorers(redirectURL)
}
