package services

import "github.com/maulanashalihin/laju-go/app/queries"

// TasbihService: tasbih digital pribadi (katalog dzikir, sesi, statistik, badge).
type TasbihService struct {
	querier *queries.Querier
}

func NewTasbihService(querier *queries.Querier) *TasbihService {
	return &TasbihService{querier: querier}
}
