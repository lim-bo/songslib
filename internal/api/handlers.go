package api

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/lim-bo/songslib/models"

	libmanager "github.com/lim-bo/songslib/internal/libManager"

	"github.com/gorilla/mux"
)

//	@Summary		Add a new song to library
//	@Description	Send json-representation of song data (name, group) and add it to database
//	@accept			json
//	@Param			song	body	models.Song	true
//	@Success		200
//	@Failure		500,	400
//	@Router			/lib/add [put]
func (api *SongsLibAPI) AddNewSong(w http.ResponseWriter, r *http.Request) {
	var s models.Song
	body := r.Body
	defer body.Close()
	{
		buf := bufio.NewReader(body)
		req := make([]byte, 0, 64)
		_, err := buf.Read(req)
		if err != nil {
			slog.Error("request body reading error")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		slog.Debug("incoming request", slog.String("from", r.RemoteAddr), slog.String("body", string(req)))
	}
	err := json.NewDecoder(body).Decode(&s)
	if err != nil {
		slog.Error("request json unmarshalling error")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sd, err := api.minfo.GetSongDetails(s)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("remote api request error: " + err.Error())
		return
	}
	slog.Info("got detailed song info", slog.String("from", r.RemoteAddr))
	slog.Debug("detailed song info to append", slog.Any("data", sd))
	err = api.lm.CreateSong(sd)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("db error: " + err.Error())
		return
	}
	slog.Info("succesful adding new song", slog.String("from", r.RemoteAddr))
	w.WriteHeader(http.StatusOK)
}

//	@Summary		Delete song from library
//	@Description	Recieves query params (name, group) and deletes choosen song, if exist
//	@Param			name	query	string	true
//	@Param			group	quey	string	true
//	@Success		200
//	@Failure		400,	500
//	@Router			/lib/remove [delete]
func (api *SongsLibAPI) DeleteSong(w http.ResponseWriter, r *http.Request) {
	pars := r.URL.Query()
	songName, groupName := pars.Get("name"), pars.Get("group")
	if songName == "" || groupName == "" {
		w.WriteHeader(http.StatusBadRequest)
		slog.Error("incoming request with lack of required params", slog.String("from", r.RemoteAddr))
		return
	}
	slog.Debug("incoming delete request", slog.String("song_name", songName), slog.String("group_name", groupName))
	err := api.lm.DeleteSong(models.Song{Name: songName, Group: groupName})
	if err != nil {
		if err == libmanager.ErrNoRows {
			w.WriteHeader(http.StatusBadRequest)
			slog.Error("no matching data to deletion", slog.String("from", r.RemoteAddr), slog.String("song_name", songName), slog.String("group_name", groupName))
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			slog.Error("db error: "+err.Error(), slog.String("from", r.RemoteAddr))
			return
		}
	}
	slog.Info("succesful deleted song", slog.String("from", r.RemoteAddr))
	w.WriteHeader(http.StatusOK)
}

//	@Summary		Get lib page with songs' data with filter via query params
//	@Description	Recieves page and limit parameters from query, which was processing in middleware,
//	@Description	and filter[like] params for selecting songs from DB
//	@Param			page			query	int		false
//	@Param			limit			query	int		true
//	@Param			name			query	string	false
//	@Param			group			query	string	false
//	@Param			release_date	query	string	false
//	@Param			lyrics			query	string	false
//	@produce		json
//	@Success		200		{array}	models.SongDetailed
//	@Failure		400,	500
//	@Router			/lib [get]
func (api *SongsLibAPI) ReadLibPage(w http.ResponseWriter, r *http.Request) {
	page := r.Context().Value("page").(int)
	limit := r.Context().Value("limit").(int)
	filter := make(map[string]string, 0)
	name := r.URL.Query().Get("name")
	if name != "" {
		filter["name"] = name
	}
	group := r.URL.Query().Get("group")
	if group != "" {
		filter["group"] = group
	}
	releaseDate := r.URL.Query().Get("releaseDate")
	if releaseDate != "" {
		filter["release_date"] = releaseDate
	}
	lyrics := r.URL.Query().Get("lyrics")
	if lyrics != "" {
		filter["lyrics"] = lyrics
	}
	slog.Debug("recieved filter settings for song pages", slog.Any("settings", filter), slog.String("from", r.RemoteAddr))
	result, err := api.lm.GetSongsPage(page, limit, filter)
	if err != nil {
		if err == libmanager.ErrBadFilterParams {
			w.WriteHeader(http.StatusBadRequest)
			slog.Error("db manager error: "+err.Error(), slog.String("from", r.RemoteAddr))
		}
	}
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("marshalling error", slog.Any("result", result))
		return
	}
	w.WriteHeader(http.StatusOK)
	slog.Info("readed songs lib page", slog.Any("data", result), slog.String("from", r.RemoteAddr))
}

//	@Summary		Get song lyrics' copultes
//	@Description	Handler recieves page and limit parameters from query and group, song name from path values, then
//	@Description	send delimited lyrics of choosen song
//	@Param			page		query	int		false
//	@Param			limit		query	int		true
//	@Param			song_name	path	string	true
//	@Param			group_name	path	string	true
//	@produce		json
//	@Success		200		{object}	models.Lyrics
//	@Failure		400,	500,		404
//	@Router			/lib/{group_name}/{song_name} [get]
func (api *SongsLibAPI) ReadSongLyricsPage(w http.ResponseWriter, r *http.Request) {
	page := r.Context().Value("page").(int)
	limit := r.Context().Value("limit").(int)
	v := mux.Vars(r)
	name, ok := v["song_name"]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	group, ok := v["group_name"]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	slog.Debug("reading lyrics request", slog.String("song_name", name), slog.String("group_name", group), slog.String("from", r.RemoteAddr))
	s, err := api.lm.GetSong(models.Song{Name: name, Group: group})
	if err != nil {
		if err == libmanager.ErrNoRows {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		slog.Error("db manager error: "+err.Error(), slog.String("from", r.RemoteAddr))
	}
	couplets := strings.Split(s.Text, "\n")
	result := make([]string, 0, 10)
	for i := page * limit; i < len(couplets) && i < limit; i++ {
		result = append(result, couplets[i])
	}
	if result == nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.Error("page index out of range", slog.String("from", r.RemoteAddr))
		return
	} else {
		err = json.NewEncoder(w).Encode(models.Lyrics{Page: page, Text: result})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			slog.Error("lyrics marshalling error"+err.Error(), slog.String("from", r.RemoteAddr))
			return
		}
		w.WriteHeader(http.StatusOK)
	}
	slog.Info("song lyrics request", slog.Any("response", models.Lyrics{Page: page, Text: result}), slog.String("from", r.RemoteAddr))
}

//	@Summary		Edit song data in database
//	@Description	Recieves new song data in json request body
//	@Description	and updates it in database
//	@accept			json
//	@Success		200
//	@Param			new_song	body	models.SongDetailed	true
//	@Failure		500,		400
//	@Router			/lib/edit [post]
func (api *SongsLibAPI) EditSongData(w http.ResponseWriter, r *http.Request) {
	var sd models.SongDetailed
	body := r.Body
	defer body.Close()
	{
		buf := bufio.NewReader(body)
		req := make([]byte, 0, 64)
		_, err := buf.Read(req)
		if err != nil {
			slog.Error("request body reading error")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		slog.Debug("incoming request", slog.String("from", r.RemoteAddr), slog.String("body", string(req)))
	}
	err := json.NewDecoder(body).Decode(&sd)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.Error("unmarshalling error: "+err.Error(), slog.String("from", r.RemoteAddr))
		return
	}
	err = api.lm.UpdateSongData(&sd)
	if err == libmanager.ErrNoRows {
		w.WriteHeader(http.StatusBadRequest)
		slog.Error("updating unexist data", slog.String("from", r.RemoteAddr))
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("db manager error: "+err.Error(), slog.String("from", r.RemoteAddr))
		return
	}
	slog.Info("updated song data", slog.Any("data", sd), slog.String("from", r.RemoteAddr))
	w.WriteHeader(http.StatusOK)
}

// Pagination middleware checks if there are page and limit query params for pagination handlers
func PaginationMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		page := r.URL.Query().Get("page")
		if page == "" {
			ctx = context.WithValue(ctx, "page", 0)
		} else {
			pageInt, err := strconv.Atoi(page)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				slog.Error("parsing page param error", slog.String("from", r.RemoteAddr), slog.String("value", page))
				return
			}
			ctx = context.WithValue(ctx, "page", pageInt)
		}

		limit := r.URL.Query().Get("limit")
		if limit == "" {
			w.WriteHeader(http.StatusBadRequest)
			slog.Error("limit param requied", slog.String("from", r.RemoteAddr))
			return
		}
		limitInt, err := strconv.Atoi(limit)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			slog.Error("parsing limit param error", slog.String("from", r.RemoteAddr), slog.String("value", page))
			return
		}
		ctx = context.WithValue(ctx, "limit", limitInt)
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}

// Applying CORS options middleware
func CORSMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "application/json")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		w.Header().Set("Access-Control-Max-Age", "20")
		w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:*")
		h.ServeHTTP(w, r)
	})
}
