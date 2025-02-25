package main

import (
	"github.com/noch-g/learn-pub-sub-starter/internal/gamelogic"
	"github.com/noch-g/learn-pub-sub-starter/internal/pubsub"
	"github.com/noch-g/learn-pub-sub-starter/internal/routing"
)

func handlerLogs() func(routing.GameLog) pubsub.ActType {
	return func(gl routing.GameLog) pubsub.ActType {
		err := gamelogic.WriteLog(gl)
		if err != nil {
			return pubsub.NackDiscard
		}
		return pubsub.Ack
	}
}
