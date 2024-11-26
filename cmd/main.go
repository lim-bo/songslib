package main

import (
	"log"
	"log/slog"
	"os"

	libmanager "github.com/lim-bo/songslib/internal/libManager"

	"github.com/lim-bo/songslib/internal/api"

	"github.com/joho/godotenv"
)

var cfg map[string]string

func init() {
	env, err := os.OpenFile("../configs/cfg.env", os.O_RDONLY, os.ModeExclusive)
	if err != nil {
		log.Fatal(err)
	}
	cfg, err = godotenv.Parse(env)
	if err != nil {
		log.Fatal(err)
	}
}

//	@title			SongsLibApi
//	@version		1.0
//	@description	API for songs library

//	@BasePath	/api/v1
func main() {
	dbCfg := libmanager.DBConfig{
		Host:     cfg["DB_HOST"],
		Port:     cfg["DB_PORT"],
		Username: cfg["DB_USER"],
		Password: cfg["DB_PASSWORD"],
		DBName:   cfg["DB_SONGSLIB_NAME"],
	}
	sv := api.New(cfg["API_VER"], dbCfg)
	if err := sv.Run(cfg["SERVER_HOST"], cfg["SERVER_PORT"]); err != nil {
		slog.Error(err.Error())
	}
}
