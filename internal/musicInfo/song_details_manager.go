package musicinfo

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/lim-bo/songslib/models"
)

type MIManager struct {
	cli *http.Client
}

func New() *MIManager {
	return &MIManager{
		cli: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

// Remote api request for detailed song information
// returns filled music info struct, error
func (m *MIManager) GetSongDetails(s models.Song) (*models.SongDetailed, error) {
	var sd models.SongDetailed
	req, err := http.NewRequest(http.MethodGet, "http://musicinfo/info", nil)
	if err != nil {
		return nil, errors.Join(errors.New("internal error"), err)
	}
	q := req.URL.Query()
	q.Add("song", s.Name)
	q.Add("group", s.Group)
	req.URL.RawQuery = q.Encode()
	resp, err := m.cli.Do(req)
	if err != nil {
		return nil, errors.Join(errors.New("remote api request error: "), err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		err = json.NewDecoder(resp.Body).Decode(&sd)
		if err != nil {
			return nil, errors.Join(errors.New("unmarshalling error: "), err)
		}
		sd.Song = s
	case http.StatusInternalServerError:
		return nil, errors.Join(errors.New("remote api internal error: "), err)
	case http.StatusBadRequest:
		return nil, errors.Join(errors.New("bad request to remote api: "), err)
	}
	return &sd, nil
}
