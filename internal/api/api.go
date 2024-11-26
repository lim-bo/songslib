package api

import (
	"net/http"

	"github.com/lim-bo/songslib/models"

	libmanager "github.com/lim-bo/songslib/internal/libManager"
	musicinfo "github.com/lim-bo/songslib/internal/musicInfo"

	"github.com/gorilla/mux"
	_ "github.com/lim-bo/songslib/cmd/docs"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type libManagerI interface {
	CreateSong(s *models.SongDetailed) error
	GetSong(s models.Song) (*models.SongDetailed, error)
	DeleteSong(s models.Song) error
	GetSongsPage(page int, elemCountPerPage int, filter map[string]string) ([]*models.SongDetailed, error)
	UpdateSongData(newData *models.SongDetailed) error
}

type MusicInfoManagerI interface {
	GetSongDetails(s models.Song) (*models.SongDetailed, error)
}

// main API class
type SongsLibAPI struct {
	ver   string
	mux   *mux.Router
	lm    libManagerI
	minfo MusicInfoManagerI
}

func New(ver string, dbCfg libmanager.DBConfig) *SongsLibAPI {
	return &SongsLibAPI{
		ver:   ver,
		mux:   mux.NewRouter(),
		lm:    libmanager.New(dbCfg),
		minfo: musicinfo.New(),
	}
}

func (api *SongsLibAPI) Run(host string, port string) error {
	paginationRoutes := api.mux.Methods(http.MethodGet).Subrouter()

	paginationRoutes.HandleFunc("api/v"+api.ver+"/lib", api.ReadLibPage)
	paginationRoutes.HandleFunc("api/v"+api.ver+"/lib/{group_name}/{song_name}", api.ReadSongLyricsPage).Methods(http.MethodGet)
	paginationRoutes.Use(PaginationMiddleware)
	api.mux.HandleFunc("api/v"+api.ver+"/lib/remove", api.DeleteSong).Methods(http.MethodDelete)
	api.mux.HandleFunc("api/v"+api.ver+"/lib/add", api.AddNewSong).Methods(http.MethodPut)
	api.mux.HandleFunc("api/v"+api.ver+"/lib/edit", api.EditSongData).Methods(http.MethodPost)
	api.mux.Handle("/swagger/", httpSwagger.WrapHandler)
	api.mux.Use(CORSMiddleware)
	return http.ListenAndServe(host+":"+port, api.mux)
}
