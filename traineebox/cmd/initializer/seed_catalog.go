package main

import (
	"context"
	"fmt"

	"traineebox/internal/platform/config"
	"traineebox/internal/platform/postgres"
	ticketsinfra "traineebox/internal/tickets/infrastructure"

	"github.com/spf13/cobra"
)

type catalogTypeSeed struct {
	Code  string
	Title string
	Tags  []catalogTagSeed
}

type catalogTagSeed struct {
	Code  string
	Title string
}

type catalogServiceSeed struct {
	Code  string
	Title string
}

var seedIncidentTypes = []catalogTypeSeed{
	{Code: "101", Title: "101"},
	{Code: "102", Title: "102"},
	{Code: "103", Title: "103"},
	{Code: "104", Title: "104"},
	{Code: "accidents_city", Title: "Аварии и происшествия в городском хозяйстве"},
	{Code: "accidents_transport", Title: "Аварии и происшествия на транспортных объектах"},
	{Code: "accidents_hydro", Title: "Аварии на гидротехнических сооружениях"},
	{Code: "accidents_hazardous", Title: "Аварии на опасных и производственных объектах"},
	{Code: "gratitude", Title: "Благодарность службам"},
	{Code: "uav", Title: "БПЛА"},
	{Code: "explosion", Title: "Взрыв"},
	{Code: "internal_call", Title: "Внутренний звонок (звонок от работников)"},
	{Code: "foreign_language", Title: "Вызов на иностранном языке"},
	{Code: "additional_call", Title: "Дополнительный звонок от заявителя"},
	{Code: "road_obstacles", Title: "Дорожные помехи"},
	{Code: "traffic_accident", Title: "ДТП"},
	{Code: "complaint", Title: "Жалоба на действие или бездействие служб"},
	{Code: "animals", Title: "Животные"},
	{Code: "consultation", Title: "Консультация"},
	{Code: "non_target_call", Title: "Нецелевой вызов"},
	{Code: "collapse", Title: "Обрушение"},
	{Code: "feedback_112", Title: "Отзыв о работе 112 Москва"},
	{Code: "call_cancel", Title: "Отмена вызова"},
	{Code: "wrong_number", Title: "Ошибочно набран номер"},
	{Code: "shift_handover", Title: "Передача дежурства"},
	{Code: "assist_services", Title: "Помощь службам"},
	{Code: "natural_disaster", Title: "Природная стихия"},
	{Code: "other", Title: "Прочие происшествия"},
	{Code: "radiation", Title: "Радиация"},
	{Code: "broken_thermometer", Title: "Разбитый градусник"},
	{Code: "child_in_danger", Title: "Ребенок в опасности"},
	{Code: "gathering", Title: "Сбор"},
	{Code: "water_accumulation", Title: "Скопление воды"},
	{Code: "fatal_outcome", Title: "Смертельный исход"},
	{Code: "social_assistance", Title: "Социальная помощь"},
	{Code: "info_101", Title: "Справка 101"},
	{Code: "info_102", Title: "Справка 102"},
	{Code: "info_103", Title: "Справка 103"},
	{Code: "info_104", Title: "Справка 104"},
	{Code: "info_gibdd", Title: "Справка ГИБДД"},
	{Code: "info_city", Title: "Справка Городское хозяйство"},
	{Code: "info_mchs", Title: "Справка МЧС"},
	{Code: "test_call", Title: "Тестовый вызов"},
	{Code: "technical_failure", Title: "Технический сбой (сбой в работе с оборудованием 112 Москва)"},
	{Code: "training", Title: "Тренировка"},
	{Code: "emergency_notification", Title: "Уведомление о ЧС"},
	{Code: "threat_explosion", Title: "Угроза взрыва/террористического акта"},
	{Code: "threat_hazardous_release", Title: "Угроза выброса опасных веществ и радиации"},
	{Code: "threat_collapse", Title: "Угроза обрушения"},
	{Code: "person_in_danger", Title: "Человек в опасности"},
	{Code: "ecological", Title: "Экологическое происшествие"},
}

var seedServices = []catalogServiceSeed{
	{Code: "fire_service", Title: "Пожарная охрана"},
	{Code: "ambulance", Title: "Скорая помощь"},
	{Code: "police", Title: "Полиция"},
}

func newSeedCatalogCmd(cfg config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "seed-catalog",
		Short: "Upsert incident types, tags and emergency services",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := seedCatalog(cmd.Context(), cfg); err != nil {
				return err
			}
			fmt.Println("catalog ready")
			return nil
		},
	}
}

func seedCatalog(ctx context.Context, cfg config.Config) error {
	pool, err := postgres.NewPool(ctx, cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	catalog := ticketsinfra.NewCatalogRepository(pool)
	for _, t := range seedIncidentTypes {
		it, err := catalog.UpsertIncidentType(ctx, t.Code, t.Title)
		if err != nil {
			return fmt.Errorf("incident type %s: %w", t.Code, err)
		}
		for _, tag := range t.Tags {
			if _, err := catalog.UpsertIncidentTag(ctx, it.ID, tag.Code, tag.Title); err != nil {
				return fmt.Errorf("tag %s/%s: %w", t.Code, tag.Code, err)
			}
		}
	}
	for _, s := range seedServices {
		if _, err := catalog.UpsertService(ctx, s.Code, s.Title); err != nil {
			return fmt.Errorf("service %s: %w", s.Code, err)
		}
	}
	return nil
}
