package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/scc749/nimbus-blog-api/config"
	"github.com/scc749/nimbus-blog-api/pkg/postgres"
	"gorm.io/gen"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Gen: config error: %s", err)
	}

	postgres, err := postgres.New(cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.DBName, cfg.Postgres.SSLMode, cfg.Postgres.TimeZone, postgres.WithMaxIdleConns(cfg.Postgres.MaxIdleConns), postgres.WithMaxOpenConns(cfg.Postgres.MaxOpenConns))

	if err != nil {
		log.Fatalf("Gen: postgres error: %s", err)
	}

	wd, _ := os.Getwd()

	g := gen.NewGenerator(gen.Config{
		OutPath:          filepath.Join(wd, "internal", "repo", "persistence", "gen", "query"),
		ModelPkgPath:     filepath.Join(wd, "internal", "repo", "persistence", "gen", "model"),
		Mode:             gen.WithDefaultQuery | gen.WithQueryInterface,
		FieldNullable:    true,
		FieldWithTypeTag: true,
	})

	g.UseDB(postgres.DB)
	// ensure generated models import gorm for gorm.DeletedAt
	g.WithImportPkgPath("gorm.io/gorm")

	log.Printf("Gen: generating models and queries")

	// g.ApplyBasic(g.GenerateAllTable()...)

	// Generate models with deleted_at mapped to gorm.DeletedAt for soft delete
	g.ApplyBasic(
		g.GenerateModel("admins", gen.FieldType("deleted_at", "gorm.DeletedAt")),
		g.GenerateModel("admin_recovery_codes"),
		g.GenerateModel("users", gen.FieldType("deleted_at", "gorm.DeletedAt")),
		g.GenerateModel("refresh_token_blacklist"),
		g.GenerateModel("categories", gen.FieldType("deleted_at", "gorm.DeletedAt")),
		g.GenerateModel("tags", gen.FieldType("deleted_at", "gorm.DeletedAt")),
		g.GenerateModel("posts", gen.FieldType("deleted_at", "gorm.DeletedAt")),
		g.GenerateModel("post_tags"),
		g.GenerateModel("post_likes"),
		g.GenerateModel("post_views"),
		g.GenerateModel("comments", gen.FieldType("deleted_at", "gorm.DeletedAt")),
		g.GenerateModel("comment_likes"),
		g.GenerateModel("site_settings", gen.FieldType("deleted_at", "gorm.DeletedAt")),
		g.GenerateModel("feedbacks", gen.FieldType("deleted_at", "gorm.DeletedAt")),
		g.GenerateModel("links", gen.FieldType("deleted_at", "gorm.DeletedAt")),
		g.GenerateModel("notifications"),
		g.GenerateModel("files"),
		g.GenerateModel("schema_migrations"),
	)

	g.Execute()
	log.Printf("Gen: generate success, wd=%s", wd)
}
