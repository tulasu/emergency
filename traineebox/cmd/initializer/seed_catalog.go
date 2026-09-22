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
	{
		Code:  "fire",
		Title: "Пожар",
		Tags: []catalogTagSeed{
			{Code: "building", Title: "Здание"},
			{Code: "vehicle", Title: "Транспорт"},
		},
	},
	{
		Code:  "medical",
		Title: "Медицина",
		Tags: []catalogTagSeed{
			{Code: "unconscious", Title: "Без сознания"},
			{Code: "injury", Title: "Травма"},
		},
	},
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
