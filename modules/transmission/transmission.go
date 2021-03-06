package transmission

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"

	yaml "gopkg.in/yaml.v2"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/transmission"
	"github.com/sirupsen/logrus"
)

// Make sure that the module is a downloader
var _ polochon.Downloader = (*Client)(nil)

func init() {
	polochon.RegisterModule(&Client{})
}

// Module constants
const (
	moduleName = "transmission"
)

// Params represents the module params
type Params struct {
	URL       string `yaml:"url"`
	CheckSSL  bool   `yaml:"check_ssl"`
	BasicAuth bool   `yaml:"basic_auth"`
	Username  string `yaml:"user"`
	Password  string `yaml:"password"`
}

// Client holds the connection with transmission
type Client struct {
	*Params
	tClient    *transmission.Client
	configured bool
}

// Init implements the module interface
func (c *Client) Init(p []byte) error {
	if c.configured {
		return nil
	}

	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return err
	}

	return c.InitWithParams(params)
}

// InitWithParams configures the module
func (c *Client) InitWithParams(params *Params) error {
	c.Params = params
	if err := c.checkConfig(); err != nil {
		return err
	}

	// Set the transmission client according to the conf
	if err := c.setTransmissionClient(); err != nil {
		return err
	}

	c.configured = true

	return nil
}

func (c *Client) checkConfig() error {
	if c.URL == "" {
		return fmt.Errorf("transmission: missing URL")
	}

	if c.BasicAuth {
		if c.Username == "" || c.Password == "" {
			return fmt.Errorf("transmission: missing authentication params")
		}
	}

	return nil
}

func (c *Client) setTransmissionClient() error {
	skipSSL := !c.CheckSSL

	// Create HTTP client with SSL configuration
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSSL},
	}
	httpClient := http.Client{Transport: tr}

	conf := transmission.Config{
		Address:    c.URL,
		User:       c.Username,
		Password:   c.Password,
		HTTPClient: &httpClient,
	}

	t, err := transmission.New(conf)
	if err != nil {
		return err
	}

	c.tClient = t

	return nil
}

// Name implements the Module interface
func (c *Client) Name() string {
	return moduleName
}

// Status implements the Module interface
func (c *Client) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusNotImplemented, nil
}

// Download implements the downloader interface
func (c *Client) Download(URL string, metadata *polochon.DownloadableMetadata, log *logrus.Entry) error {
	t, err := c.tClient.Add(URL)
	if err != nil {
		return err
	}

	labels := labels(metadata)
	if labels == nil {
		return nil
	}

	return t.Set(transmission.SetTorrentArg{
		Labels: labels,
	})
}

// List implements the downloader interface
func (c *Client) List() ([]polochon.Downloadable, error) {
	torrents, err := c.tClient.GetTorrents()
	if err != nil {
		return nil, err
	}

	var res []polochon.Downloadable
	for _, t := range torrents {
		res = append(res, Torrent{
			T: t,
		})
	}

	return res, nil
}

// Remove implements the downloader interface
func (c *Client) Remove(d polochon.Downloadable) error {
	// Get infos from the torrent
	tInfos := d.Infos()
	if tInfos == nil {
		return fmt.Errorf("transmission: got nil Infos")
	}

	// Get the torrentID needed to delete the torrent
	if tInfos.ID == "" {
		return fmt.Errorf("transmission: problem when getting torrentID in Remove")
	}

	torrentID, err := strconv.Atoi(tInfos.ID)
	if err != nil {
		return fmt.Errorf("transmission: the id is not a int")
	}

	// Delete the torrent and the data
	return c.tClient.RemoveTorrents([]*transmission.Torrent{{ID: torrentID}}, false)
}

// Torrent represents a Torrent
type Torrent struct {
	T *transmission.Torrent
}

// Infos prints the Torrent status
func (t Torrent) Infos() *polochon.DownloadableInfos {
	if t.T == nil {
		return nil
	}
	isFinished := false

	// Check that the torrent is finished
	if t.T.PercentDone == 1 {
		isFinished = true
	}

	// Add the filePaths
	var filePaths []string
	if t.T.Files != nil {
		for _, f := range *t.T.Files {
			filePaths = append(filePaths, f.Name)
		}
	}

	return &polochon.DownloadableInfos{
		ID:             strconv.Itoa(t.T.ID),
		DownloadRate:   t.T.RateDownload,
		DownloadedSize: int(t.T.DownloadedEver),
		UploadedSize:   int(t.T.UploadedEver),
		FilePaths:      filePaths,
		IsFinished:     isFinished,
		Name:           t.T.Name,
		PercentDone:    float32(t.T.PercentDone) * 100,
		Ratio:          float32(t.T.UploadRatio),
		TotalSize:      int(t.T.SizeWhenDone),
		UploadRate:     t.T.RateUpload,
		Metadata:       metadata(t.T.Labels),
	}
}
